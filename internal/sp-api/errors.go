package sp_api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
)

// recordError is a generic function that processes SP-API responses and handles error cases.
// It takes a pointer to a response struct of any type and an error as input.
// The function examines the HTTP response status code and, for status codes >= 400,
// attempts to marshal the corresponding JSON error structure into a formatted error message.
//
// Parameters:
//   - resp: A pointer to the response struct of type T
//   - err: An error object from the initial API call
//
// Returns:
//   - (*T, error): Returns the original response pointer and either:
//   - nil error if status code < 400
//   - formatted error message containing URL, status code, and error details for status code >= 400
//   - original error if initial error was non-nil
//   - marshalling error if JSON error structure cannot be marshalled
//
// Example:
//
//	func (a *Client) PutListingsItem(ctx context.Context, sellerId string, sku string, params *listings.PutListingsItemParams, body listings.PutListingsItemJSONRequestBody) (*listings.PutListingsItemResp, error) {
//		return recordError(a.GetListingsService().PutListingsItemWithResponse(ctx, sellerId, sku, params, body, a.WithAuth(), a.WithRateLimit(CreateListingRLKey)))
//	}
//
// https://go.dev/play/p/jGQbR-Ri6Wq
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
