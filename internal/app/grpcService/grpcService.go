package grpcService

import (
	"context"
	"net/http"
	"strings"

	"github.com/jackc/pgerrcode"
	"github.com/trunov/go-shortener/internal/app/encryption"
	"github.com/trunov/go-shortener/internal/app/handler"
	"github.com/trunov/go-shortener/internal/app/util"
	pb "github.com/trunov/go-shortener/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey struct {
	name string
}

var userIDKey = &contextKey{"user-id"}

type grpcServer struct {
	pb.UnimplementedUrlShortenerServer
	// should be business logic not handler
	handler *handler.Handler
}

func NewGrpcServer(h *handler.Handler) grpcServer {
	return grpcServer{
		handler: h,
	}
}

func (s *grpcServer) ShortenLink(ctx context.Context, req *pb.ShortenRequest) (*pb.ShortenResponse, error) {
	userID := ctx.Value(userIDKey).(string)

	shortenedURL, statusCode, err := s.handler.ProcessShortenLink(req.GetUrl(), userID)

	if err != nil {
		grpcErr := status.Error(convertToGrpcStatusCode(statusCode), err.Error())

		if shortenedURL != "" {
			return &pb.ShortenResponse{
				ShortenedUrl: shortenedURL,
			}, grpcErr
		}

		return nil, grpcErr
	}

	return &pb.ShortenResponse{
		ShortenedUrl: shortenedURL,
	}, nil
}

func (s *grpcServer) GetURLLink(ctx context.Context, req *pb.GetURLRequest) (*pb.GetURLResponse, error) {
	url, isDeleted, err := s.handler.RetrieveURL(ctx, req.GetKey())

	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}

	if isDeleted {
		return nil, status.Errorf(codes.NotFound, "URL is deleted")
	}

	return &pb.GetURLResponse{Url: url}, nil
}

func (s *grpcServer) GetUrlsByUserID(ctx context.Context, req *pb.GetUrlsByUserIDRequest) (*pb.GetUrlsByUserIDResponse, error) {
	userID := ctx.Value(userIDKey).(string)

	allURLsByUserID, err := s.handler.RetrieveURLsByUserID(ctx, userID)

	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if len(allURLsByUserID) == 0 {
		return &pb.GetUrlsByUserIDResponse{}, nil
	}

	response := &pb.GetUrlsByUserIDResponse{
		Urls: make([]*pb.UrlResponse, len(allURLsByUserID)),
	}

	for i, urlMapping := range allURLsByUserID {
		response.Urls[i] = &pb.UrlResponse{
			ShortUrl:    urlMapping.ShortURL,
			OriginalUrl: urlMapping.OriginalURL,
		}
	}

	return response, nil
}

func (s *grpcServer) DeleteUrls(ctx context.Context, req *pb.DeleteUrlsRequest) (*pb.DeleteUrlsResponse, error) {
	userID := ctx.Value(userIDKey).(string)

	if err := s.handler.ProcessDeletion(ctx, req.GetUrls(), userID); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &pb.DeleteUrlsResponse{}, nil
}

func (s *grpcServer) ShortenLinksInBatch(ctx context.Context, req *pb.ShortenLinksInBatchRequest) (*pb.ShortenLinksInBatchResponse, error) {
	userID := ctx.Value(userIDKey).(string)

	var batchReq []handler.BatchRequest
	for _, br := range req.GetBatchRequest() {
		batchReq = append(batchReq, handler.BatchRequest{
			CorrelationID: br.GetCorrelationId(),
			OriginalURL:   br.GetOriginalUrl(),
		})
	}

	batchRes, k, err := s.handler.ProcessBatchShortening(ctx, batchReq, userID)
	var pbBatchResponses []*pb.BatchResponse

	if err != nil {
		if strings.Contains(err.Error(), pgerrcode.UniqueViolation) || strings.Contains(err.Error(), "found entry") {
			return &pb.ShortenLinksInBatchResponse{
				BatchResponse: pbBatchResponses,
				ConflictKey:   k,
			}, status.Error(codes.AlreadyExists, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	for _, br := range batchRes {
		pbBr := &pb.BatchResponse{
			CorrelationId: br.CorrelationID,
			ShortUrl:      br.ShortURL,
			OriginalUrl:   br.OriginalURL,
			UserId:        br.UserID,
		}
		pbBatchResponses = append(pbBatchResponses, pbBr)
	}

	return &pb.ShortenLinksInBatchResponse{BatchResponse: pbBatchResponses}, nil
}

func (s *grpcServer) GetInternalStats(ctx context.Context, req *pb.GetInternalStatsRequest) (*pb.GetInternalStatsResponse, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		ipStr := md.Get("x-real-ip")
		if len(ipStr) > 0 {
			if !s.handler.CheckClientIP(ipStr[0]) {
				return nil, status.Error(codes.PermissionDenied, "IP address not allowed")
			}
		}
	}

	stats, err := s.handler.RetrieveInternalStats(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	response := &pb.GetInternalStatsResponse{
		Urls:  int32(stats.Urls),
		Users: int32(stats.Users),
	}

	return response, nil
}

func AuthInterceptor(key []byte) grpc.UnaryServerInterceptor {
	encryptor := encryption.NewEncryptor(key)

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		// I am not sure if we need to check it
		if !ok {
			return nil, status.Errorf(codes.Internal, "missing metadata")
		}

		// maybe different name for ctx ?
		ctx, userID, err := handleUserID(md, encryptor, ctx)
		if err != nil {
			return nil, status.Errorf(codes.PermissionDenied, "")
		}

		newCtx := context.WithValue(ctx, userIDKey, userID)

		return handler(newCtx, req)
	}
}

func handleUserID(md metadata.MD, encryptor *encryption.Encryptor, ctx context.Context) (context.Context, string, error) {
	values := md.Get(userIDKey.name)
	var userID string
	var err error

	if len(values) > 0 {
		encodedUserID := values[0]

		userIDBytes, err := encryptor.Decode(encodedUserID)
		if err != nil {
			return ctx, "", err
		}
		userID = string(userIDBytes)
	}

	if userID == "" {
		userID, err = util.GenerateRandomUserID()
		if err != nil {
			return ctx, "", err
		}

		encoded, err := encryptor.Encode([]byte(userID))
		if err != nil {
			return ctx, "", err
		}

		outMD := metadata.Pairs(userIDKey.name, encoded)
		newCtx := metadata.NewOutgoingContext(ctx, outMD)

		err = grpc.SendHeader(newCtx, outMD)
		if err != nil {
			return newCtx, "", status.Errorf(codes.Internal, "failed to send metadata: %v", err)
		}

		md.Set(userIDKey.name, encoded)

		return newCtx, userID, nil
	}

	return ctx, userID, nil
}

func convertToGrpcStatusCode(httpStatusCode int) codes.Code {
	switch httpStatusCode {
	case http.StatusOK, http.StatusCreated:
		return codes.OK
	case http.StatusBadRequest:
		return codes.InvalidArgument
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusNotFound:
		return codes.NotFound
	case http.StatusInternalServerError:
		return codes.Internal
	case http.StatusConflict:
		return codes.AlreadyExists
	default:
		return codes.Unknown
	}
}
