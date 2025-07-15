package http

import "net/http"

type (
	// ResponseData статистика по ответу.
	ResponseData struct {
		Status int
		Size   int
		Body   []byte
		Err    error
	}

	// Writer структура для вывода данных.
	Writer struct {
		http.ResponseWriter
		responseData *ResponseData
	}
)

// Write запись ответа.
func (w *Writer) Write(b []byte) (int, error) {
	w.responseData.Body = append(w.responseData.Body, b...)
	w.responseData.Size += len(b)
	return w.ResponseWriter.Write(b)
}

// WriteHeader устанавливает статус.
func (w *Writer) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.responseData.Status = statusCode
}

// ResponseData данные по ответу.
func (w *Writer) ResponseData() *ResponseData {
	return w.responseData
}

// NewWriter конструктор.
func NewWriter(w http.ResponseWriter, d *ResponseData) *Writer {
	return &Writer{
		ResponseWriter: w,
		responseData:   d,
	}
}
