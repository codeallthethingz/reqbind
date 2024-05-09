package reqbind

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

// UnmarshalBody is a custom unmarshaler that will check for required fields
// and throw an error if the field is missing
func UnmarshalBody(r *http.Request, v interface{}) error {
	bodyBytes, err := getBodyBytes(r)
	if err != nil {
		return err
	}

	if len(bodyBytes) == 0 {
		return nil
	}

	if err := json.Unmarshal(bodyBytes, v); err != nil {
		return err
	}

	return checkMetadata(v)
}

func UnmarshalQuery(r *http.Request, v interface{}) error {
	qMap := make(map[string]interface{})
	for k, value := range r.URL.Query() {
		if len(value) == 0 || value[0] == "" {
			continue
		}
		qMap[strings.ToLower(k)] = coerceToType(value[0])
	}

	b, err := json.Marshal(qMap)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return err
	}

	return checkMetadata(v)
}

func UnmarshalURLParams(r *http.Request, v interface{}) error {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return fmt.Errorf("no route context")
	}
	queryMap := make(map[string]string)

	for i, key := range rctx.URLParams.Keys {
		queryMap[key] = rctx.URLParams.Values[i]
	}

	j, err := json.Marshal(queryMap)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(j, v); err != nil {
		return err
	}

	return checkMetadata(v)
}

func getBodyBytes(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}

	return io.ReadAll(r.Body)
}

func coerceToType(value string) interface{} {
	if i, err := strconv.Atoi(value); err == nil {
		return i
	}
	if b, err := strconv.ParseBool(value); err == nil {
		return b
	}

	if !strings.HasPrefix(value, ".") {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}

	if unescaped, err := url.QueryUnescape(value); err != nil {
		return value
	} else {
		return unescaped
	}
}

func checkMetadata(v interface{}) error {
	// get the type of the object
	t := reflect.TypeOf(v).Elem()

	// iterate through the fields and check for required
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		// if the field is required, check for the zero value
		if f.Tag.Get("required") == "true" {
			reflectValue := reflect.ValueOf(v).Elem()
			// deal with : <invalid reflect.Value>
			if reflectValue.Kind() == reflect.Invalid {
				return fmt.Errorf("field %s is required", f.Name)
			}

			// get the value of the field
			value := reflect.ValueOf(v).Elem().FieldByName(f.Name)
			// if the value is the zero value and not a boolean
			if value.IsZero() && f.Type.Kind() != reflect.Bool {
				return fmt.Errorf("field %s is required", f.Name)
			}
			// if it's a pointer and nil then throw an error
			if f.Type.Kind() == reflect.Ptr && value.IsNil() {
				return fmt.Errorf("field %s is required", f.Name)
			}
		}

		// if the field has a max-length, check the length
		if f.Tag.Get("max-length") != "" {
			// get the value of the field
			value := reflect.ValueOf(v).Elem().FieldByName(f.Name)
			// conver the tag max-length to an int
			if maxLengthInt, err := strconv.Atoi(f.Tag.Get("max-length")); err != nil {
				return fmt.Errorf("field %s has invalid max-length", f.Name)
			} else {
				// if the value is the zero value, then throw an error
				if len(value.String()) > maxLengthInt {
					// truncate
					value.SetString(value.String()[0:maxLengthInt])
				}
			}
		}

		// if the field has a trimlower, trim and lowercase
		if f.Tag.Get("trimlower") == "true" {
			// get the value of the field
			value := reflect.ValueOf(v).Elem().FieldByName(f.Name)
			// trim and lowercase
			value.SetString(strings.TrimSpace(strings.ToLower(value.String())))
		}

		// if the field has a validate, get the validation type (email, phone) and validate
		if f.Tag.Get("validate") != "" {
			vType := f.Tag.Get("validate")

			// get the value of the field
			value := reflect.ValueOf(v).Elem().FieldByName(f.Name)

			// validate the value
			if vType == "email" {
				if err := validateEmail(value.String(), vType); err != nil {
					return fmt.Errorf("field %s is invalid: %s", f.Name, err)
				}
			} else if vType == "phone" {
				if newValue, err := validatePhone(value.String()); err != nil {
					return fmt.Errorf("field %s is invalid: %s", f.Name, err)
				} else {
					value.SetString(newValue)
				}
			} else {
				return fmt.Errorf("field %s has invalid validation type", f.Name)
			}

		}

		// if this is a nested pointer to a struct, then call checkMetadata on the nested struct
		if f.Type.Kind() == reflect.Ptr && f.Type.Elem().Kind() == reflect.Struct {
			if err := checkMetadata(reflect.ValueOf(v).Elem().FieldByName(f.Name).Interface()); err != nil {
				return err
			}
		}

		// if it's a nested struct then call checkMetadata on the nested struct,
		if f.Type.Kind() == reflect.Struct {
			if err := checkMetadata(reflect.ValueOf(v).Elem().FieldByName(f.Name).Addr().Interface()); err != nil {
				return err
			}
		}

	}
	return nil
}

func validatePhone(value string) (string, error) {
	// replace all the spaces with nothing.
	// replace any alpha characters with nothing except x
	// if the length is not 10 or greater, return an error

	newValue := strings.ReplaceAll(value, " ", "")
	newValue = strings.ReplaceAll(newValue, "(", "")
	newValue = strings.ReplaceAll(newValue, ")", "")
	newValue = strings.ReplaceAll(newValue, "-", "")
	newValue = strings.Map(func(r rune) rune {
		if r == 'x' || r == '+' || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, newValue)

	if len(newValue) < 10 {
		return "", fmt.Errorf("invalid phone number")
	}

	return newValue, nil
}

func validateEmail(value string, validationType string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	switch validationType {
	case "email":
		if !emailRegex.MatchString(value) {
			return fmt.Errorf("invalid email address")
		}
	}
	return nil
}
