package sp_api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
)

// https://go.dev/play/p/jGQbR-Ri6WQ
func recordError[T any](resp *T, err error) (*T, error) {
	if err != nil {
		return nil, err
	}
	ps := reflect.ValueOf(&resp)
	sptr := ps.Elem()
	s := reflect.Indirect(sptr)
	http_resp := s.FieldByName("HTTPResponse").Interface().(*http.Response)
	if http_resp.StatusCode < 400 {
		return resp, err
	}
	json_struct := s.FieldByName(fmt.Sprintf("JSON%d", http_resp.StatusCode)).Interface()
	var errMarshalled []byte
	if errMarshalled, err = json.Marshal(json_struct); err != nil {
		return nil, fmt.Errorf("error while marshalling error json struct: %w", err)
	}
	return resp, fmt.Errorf("<< %d %s", http_resp.StatusCode, string(errMarshalled))
}
