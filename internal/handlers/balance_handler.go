package handlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"platform/backend/internal/middleware"
	"platform/backend/internal/models"
	"platform/backend/internal/service"
)


const maxReceiptSize = 20 << 20

var allowedReceiptTypes = map[string]bool{
	"application/pdf": true,
	"image/png":       true,
	"image/jpeg":      true,
	"image/heic":      true,
	"image/heif":      true,
}

type BalanceService interface {
	GetBalance(ctx context.Context, userID any) (models.Balance, error)
	CreditFromReceipt(ctx context.Context, userID any, bucket, fileName, contentType string, data []byte) (models.Receipt, models.Balance, error)
	ListTransactions(ctx context.Context, userID any, limit, offset int) ([]models.BalanceTransaction, error)
	ListReceipts(ctx context.Context, userID any, bucket string, limit, offset int) ([]models.Receipt, error)
	RejectReceipt(ctx context.Context, receiptID int64, comment string) (models.Balance, error)
	AdjustBalance(ctx context.Context, userID any, amountMinor int64, currency, comment string) (models.Balance, error)
	ChargeGame(ctx context.Context, gameID int64, userIDs []int64) (models.GameCharge, error)

	ListAllReceipts(ctx context.Context, status, bucket string, limit, offset int) ([]models.Receipt, int64, error)
	GetUserBalance(ctx context.Context, userID any) (models.Balance, error)
	ListUserTransactions(ctx context.Context, userID any, limit, offset int) ([]models.BalanceTransaction, error)
	ListUserReceipts(ctx context.Context, userID any, bucket string, limit, offset int) ([]models.Receipt, error)
}

func GetBalance(balanceService BalanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := middleware.CurrentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		balance, err := balanceService.GetBalance(c.Request.Context(), user.ID)
		if err != nil {
			middleware.Logger(c).Error("load balance failed", slog.Any("error", err), slog.Any("user_id", user.ID))
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load balance"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"balance": balance})
	}
}

func ListBalanceTransactions(balanceService BalanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := middleware.CurrentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		limit, offset := paginationParams(c)

		transactions, err := balanceService.ListTransactions(c.Request.Context(), user.ID, limit, offset)
		if err != nil {
			middleware.Logger(c).Error("list balance transactions failed", slog.Any("error", err), slog.Any("user_id", user.ID))
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load balance transactions"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"transactions": transactions})
	}
}

func ListReceipts(balanceService BalanceService, bucket string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := middleware.CurrentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		limit, offset := paginationParams(c)

		receipts, err := balanceService.ListReceipts(c.Request.Context(), user.ID, bucket, limit, offset)
		if err != nil {
			middleware.Logger(c).Error("list receipts failed", slog.Any("error", err), slog.Any("user_id", user.ID))
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load receipts"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"receipts": receipts})
	}
}

func UploadReceipt(balanceService BalanceService, bucket string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := middleware.CurrentUser(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxReceiptSize+1<<10)

		fileHeader, err := c.FormFile("receipt")
		if err != nil {
			middleware.Logger(c).Warn("receipt form file missing or invalid", slog.Any("error", err))
			c.JSON(http.StatusBadRequest, gin.H{"message": "receipt file is required (multipart/form-data, field name \"receipt\") and must not exceed 20MB"})
			return
		}

		contentType := fileHeader.Header.Get("Content-Type")
		if !allowedReceiptTypes[contentType] {
			c.JSON(http.StatusBadRequest, gin.H{"message": "unsupported receipt file type, expected pdf, png, jpeg or heic"})
			return
		}

		file, err := fileHeader.Open()
		if err != nil {
			middleware.Logger(c).Error("open receipt file failed", slog.Any("error", err), slog.Any("user_id", user.ID))
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to read receipt file"})
			return
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			middleware.Logger(c).Warn("read receipt file failed", slog.Any("error", err), slog.Any("user_id", user.ID))
			c.JSON(http.StatusBadRequest, gin.H{"message": "failed to read receipt file"})
			return
		}

		receipt, balance, err := balanceService.CreditFromReceipt(c.Request.Context(), user.ID, bucket, fileHeader.Filename, contentType, data)
		if err != nil {
			writeReceiptError(c, err, user.ID)
			return
		}

		c.JSON(http.StatusCreated, gin.H{"receipt": receipt, "balance": balance})
	}
}


func ListAllReceipts(balanceService BalanceService, bucket string) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, offset := paginationParams(c)
		status := c.Query("status")

		receipts, total, err := balanceService.ListAllReceipts(c.Request.Context(), status, bucket, limit, offset)
		if err != nil {
			if errors.Is(err, service.ErrInvalidReceiptStatus) {
				c.JSON(http.StatusBadRequest, gin.H{
					"message": "status must be one of: " + strings.Join(service.ReceiptStatuses, ", "),
				})
				return
			}
			middleware.Logger(c).Error("list all receipts failed", slog.Any("error", err), slog.String("status", status))
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to load receipts"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"receipts": receipts, "total": total})
	}
}

func GetUserBalanceByID(balanceService BalanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := pathUserID(c)
		if !ok {
			return
		}

		balance, err := balanceService.GetUserBalance(c.Request.Context(), userID)
		if err != nil {
			writeAdminUserError(c, err, "load user balance failed", "failed to load user balance", userID)
			return
		}

		c.JSON(http.StatusOK, gin.H{"balance": balance})
	}
}

func ListUserBalanceTransactions(balanceService BalanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := pathUserID(c)
		if !ok {
			return
		}
		limit, offset := paginationParams(c)

		transactions, err := balanceService.ListUserTransactions(c.Request.Context(), userID, limit, offset)
		if err != nil {
			writeAdminUserError(c, err, "list user balance transactions failed", "failed to load balance transactions", userID)
			return
		}

		c.JSON(http.StatusOK, gin.H{"transactions": transactions})
	}
}

func ListUserReceipts(balanceService BalanceService, bucket string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := pathUserID(c)
		if !ok {
			return
		}
		limit, offset := paginationParams(c)

		receipts, err := balanceService.ListUserReceipts(c.Request.Context(), userID, bucket, limit, offset)
		if err != nil {
			writeAdminUserError(c, err, "list user receipts failed", "failed to load receipts", userID)
			return
		}

		c.JSON(http.StatusOK, gin.H{"receipts": receipts})
	}
}

type adjustBalanceRequest struct {
	UserID      int64  `json:"user_id"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
	Comment     string `json:"comment"`
}

func AdjustBalance(balanceService BalanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req adjustBalanceRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "user_id and amount_minor are required"})
			return
		}
		if req.UserID <= 0 || req.AmountMinor == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"message": "user_id must be positive and amount_minor must not be zero"})
			return
		}

		balance, err := balanceService.AdjustBalance(c.Request.Context(), req.UserID, req.AmountMinor, req.Currency, req.Comment)
		if err != nil {
			if errors.Is(err, service.ErrBalanceCurrencyMismatch) {
				c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "currency does not match balance currency"})
				return
			}
			writeAdminUserError(c, err, "adjust balance failed", "failed to adjust balance", req.UserID)
			return
		}

		c.JSON(http.StatusOK, gin.H{"balance": balance})
	}
}

type chargeGameRequest struct {
	GameID  int64   `json:"game_id"`
	UserIDs []int64 `json:"user_ids"`
}

func ChargeGame(balanceService BalanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req chargeGameRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "game_id and user_ids are required"})
			return
		}

		charge, err := balanceService.ChargeGame(c.Request.Context(), req.GameID, req.UserIDs)
		if err != nil {
			var notFoundErr *service.UsersNotFoundError
			switch {
			case errors.Is(err, service.ErrInvalidGameCharge):
				c.JSON(http.StatusBadRequest, gin.H{
					"message": "game_id must be positive, user_ids must be non-empty positive ids and not exceed the game price in minor units",
				})
			case errors.Is(err, service.ErrGameNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "game not found"})
			case errors.Is(err, service.ErrGameAlreadyCharged):
				c.JSON(http.StatusConflict, gin.H{"message": "game already charged"})
			case errors.As(err, &notFoundErr):
				c.JSON(http.StatusNotFound, gin.H{"message": "users not found", "user_ids": notFoundErr.UserIDs})
			case errors.Is(err, service.ErrBalanceCurrencyMismatch):
				c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "currency does not match balance currency"})
			default:
				middleware.Logger(c).Error("charge game failed", slog.Any("error", err), slog.Int64("game_id", req.GameID), slog.Any("user_ids", req.UserIDs))
				c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to charge game"})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{"charge": charge})
	}
}

type rejectReceiptRequest struct {
	Comment string `json:"comment"`
}

func RejectReceipt(balanceService BalanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		receiptID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || receiptID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"message": "receipt id must be a positive integer"})
			return
		}

		var req rejectReceiptRequest
		if c.Request.ContentLength > 0 {
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
				return
			}
		}

		balance, err := balanceService.RejectReceipt(c.Request.Context(), receiptID, req.Comment)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrReceiptNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "receipt not found"})
			case errors.Is(err, service.ErrReceiptAlreadyReversed):
				c.JSON(http.StatusConflict, gin.H{"message": "receipt already reversed"})
			default:
				middleware.Logger(c).Error("reject receipt failed", slog.Any("error", err), slog.Int64("receipt_id", receiptID))
				c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to reject receipt"})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{"balance": balance})
	}
}

func writeReceiptError(c *gin.Context, err error, userID any) {
	switch {
	case errors.Is(err, service.ErrReceiptDuplicate):
		middleware.Logger(c).Warn("duplicate receipt upload", slog.Any("error", err), slog.Any("user_id", userID))
		c.JSON(http.StatusConflict, gin.H{"message": "receipt already uploaded"})
	case errors.Is(err, service.ErrReceiptNotRecognized), errors.Is(err, service.ErrReceiptBadFile):
		middleware.Logger(c).Warn("receipt not recognized", slog.Any("error", err), slog.Any("user_id", userID))
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "failed to recognize receipt amount"})
	case errors.Is(err, service.ErrBalanceCurrencyMismatch):
		middleware.Logger(c).Warn("receipt currency mismatch", slog.Any("error", err), slog.Any("user_id", userID))
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "receipt currency does not match balance currency"})
	case errors.Is(err, service.ErrParsingUnavailable):
		middleware.Logger(c).Error("parsing service unavailable", slog.Any("error", err), slog.Any("user_id", userID))
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "receipt parsing service unavailable"})
	default:
		middleware.Logger(c).Error("credit receipt failed", slog.Any("error", err), slog.Any("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to process receipt"})
	}
}


func pathUserID(c *gin.Context) (int64, bool) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "user_id must be a positive integer"})
		return 0, false
	}
	return userID, true
}

func writeAdminUserError(c *gin.Context, err error, logMessage, clientMessage string, userID any) {
	if errors.Is(err, service.ErrUserNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"message": "user not found"})
		return
	}

	middleware.Logger(c).Error(logMessage, slog.Any("error", err), slog.Any("target_user_id", userID))
	c.JSON(http.StatusInternalServerError, gin.H{"message": clientMessage})
}

func paginationParams(c *gin.Context) (limit, offset int) {
	limit, _ = strconv.Atoi(c.Query("limit"))
	offset, _ = strconv.Atoi(c.Query("offset"))
	return limit, offset
}
