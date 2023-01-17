package fiberrest

import (
	"encoding/json"
	"encoding/xml"
	"net/http"

	"github.com/golangid/candi/candihelper"
	"github.com/golangid/candi/candishared"
	"github.com/valyala/fasthttp"
)

// HTTPResponse default candi http response format
type HTTPResponse struct {
	Success bool        `json:"success"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Meta    interface{} `json:"meta,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
}

// NewHTTPResponse for create common response
func NewHTTPResponse(code int, message string, params ...interface{}) *HTTPResponse {
	commonResponse := new(HTTPResponse)

	for _, param := range params {
		switch val := param.(type) {
		case *candishared.Meta, candishared.Meta:
			commonResponse.Meta = val
		case candihelper.MultiError:
			commonResponse.Errors = val.ToMap()
		case error:
			commonResponse.Errors = candihelper.NewMultiError().Append("detail", val).ToMap()
		default:
			commonResponse.Data = param
		}
	}

	if code < http.StatusBadRequest {
		commonResponse.Success = true
	}
	commonResponse.Code = code
	commonResponse.Message = message
	return commonResponse
}

// JSON for set http JSON response (Content-Type: application/json) with parameter is http response writer
func (resp *HTTPResponse) JSON(w *fasthttp.Response) error {
	w.Header.Set(candihelper.HeaderContentType, candihelper.HeaderMIMEApplicationJSON)
	w.SetStatusCode(resp.Code)
	return json.NewEncoder(w.BodyWriter()).Encode(resp)
}

// XML for set http XML response (Content-Type: application/xml)
func (resp *HTTPResponse) XML(w *fasthttp.Response) error {
	w.Header.Set(candihelper.HeaderContentType, candihelper.HeaderMIMEApplicationXML)
	w.SetStatusCode(resp.Code)
	return xml.NewEncoder(w.BodyWriter()).Encode(resp)
}
