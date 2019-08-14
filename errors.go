package yandexmapclient

import (
	"fmt"
	"strconv"
)

type ErrorTypes int

const (
	ErrorUnknown ErrorTypes = iota
	ErrorWrongStatusCode
	ErrorEmptyToken
)

// YandexClientError is an error returned by client
type YandexClientError struct {
	Message string
	code    ErrorTypes
	cause   error
}

func (err YandexClientError) Error() string {
	return err.Message
}

func (err YandexClientError) Cause() error {
	if err.cause != nil {
		return err.cause
	}

	return err
}

func CheckErrorType(err error, EType ErrorTypes) bool {
	yerr, ok := err.(YandexClientError)
	if !ok {
		return false
	}

	return yerr.code == EType
}

func ExtractErrorType(err error) ErrorTypes {
	yerr, ok := err.(YandexClientError)
	if !ok {
		return ErrorUnknown
	}

	return yerr.code
}

func NewWrongStatusCodeError(gotCode int) YandexClientError {
	return YandexClientError{
		Message: fmt.Sprintf("wrong status code, got %d", gotCode),
		code:    ErrorWrongStatusCode,
	}
}

func NewEmptyTokenError() YandexClientError {
	return YandexClientError{
		Message: "empty token from yandex",
		code:    ErrorEmptyToken,
	}
}

// YandexMapError reports error from service
type YandexMapError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *YandexMapError) Error() string {
	return "code " + strconv.Itoa(e.Code) + " and message: " + e.Message
}
