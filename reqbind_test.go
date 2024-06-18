package reqbind

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestNestedStruct(t *testing.T) {
	b := &struct {
		Value struct {
			InnerValue string `required:"true"`
		}
	}{}

	request, err := http.NewRequest("GET", "/", io.NopCloser(bytes.NewReader([]byte(`{"value":{"innerValue":"aoeu"}}`))))
	require.NoError(t, err)
	require.NoError(t, UnmarshalBody(request, b))
	require.Equal(t, "aoeu", b.Value.InnerValue)

	b = &struct {
		Value struct {
			InnerValue string `required:"true"`
		}
	}{}
	request, err = http.NewRequest("GET", "/", io.NopCloser(bytes.NewReader([]byte(`{"value":{}}`))))
	require.NoError(t, err)
	require.Error(t, UnmarshalBody(request, b))

	b2 := &struct {
		Value *struct {
			InnerValue *string
		}
	}{}
	request, err = http.NewRequest("GET", "/", io.NopCloser(bytes.NewReader([]byte(`{}`))))
	require.NoError(t, err)
	require.NoError(t, UnmarshalBody(request, b2))
}

func TestBadBody(t *testing.T) {
	k := &struct {
		Value string `required:"true"`
	}{}
	badBody := io.NopCloser(bytes.NewReader([]byte("aoeu")))
	request, err := http.NewRequest("GET", "/", badBody)
	require.NoError(t, err)
	require.Error(t, UnmarshalBody(request, k))
}

func TestUnmarshalURLParamsToStruct(t *testing.T) {
	k := &struct {
		Value string `required:"true" trimlower:"true"`
	}{}

	// create chi request
	r := chi.NewRouter()
	r.Get("/{value}", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, UnmarshalURLParams(r, k))
		require.Equal(t, "aoeu", k.Value)
	})
	req, err := http.NewRequest("GET", "/AOEU", nil)
	require.Error(t, UnmarshalURLParams(req, k), "should fail because of required chi context")
	require.NoError(t, err)
	r.ServeHTTP(nil, req)
}

func TestUnknownValidationType(t *testing.T) {
	k := &struct {
		Value string `required:"true" validate:"aoeu"`
	}{}

	request, err := http.NewRequest("GET", "/?value=aoeu", nil)
	require.NoError(t, err)
	require.Error(t, UnmarshalQuery(request, k))
}

func TestPhone(t *testing.T) {
	tests := []struct {
		value      string
		expected   string
		shouldPass bool
	}{
		{value: "aoeu", shouldPass: false},
		{value: "'1234567890'", expected: "1234567890", shouldPass: true},
		{value: "123-456-7890", expected: "1234567890", shouldPass: true},
		{value: "123.456.7890", expected: "1234567890", shouldPass: true},
		{value: "(123) 456-7890", expected: "1234567890", shouldPass: true},
		{value: "123 456 7890", expected: "1234567890", shouldPass: true},
		{value: "123-456-7890 x1234", expected: "1234567890x1234", shouldPass: true},
		{value: "123-456-7890 ext1234", expected: "1234567890x1234", shouldPass: true},
		{value: "123-456-7890 x 1234", expected: "1234567890x1234", shouldPass: true},
		{value: "123-456-7890 ext 1234", expected: "1234567890x1234", shouldPass: true},
		{value: "+1 123-456-7890", expected: "+11234567890", shouldPass: true},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			k := &struct {
				Value string `required:"true" validate:"phone"`
			}{}

			runReqTests(t, k, test.value, !test.shouldPass, true)
			if !test.shouldPass {
				return
			}
			require.Equal(t, test.expected, k.Value, fmt.Sprintf("Phone: %s", test.value))
		})
	}
}

func TestEscape(t *testing.T) {
	k := &struct {
		Value string `required:"true"`
	}{}

	request, err := http.NewRequest("GET", "/?value=a+b", nil)
	require.NoError(t, err)
	require.NoError(t, UnmarshalQuery(request, k))
	require.Equal(t, "a b", k.Value)
}

func TestNils(t *testing.T) {
	k := &struct {
		ID *int
	}{}

	request, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)
	if err := UnmarshalQuery(request, k); err != nil {
		require.NoError(t, err)
	}
	if err := UnmarshalBody(request, k); err != nil {
		require.NoError(t, err)
	}
}

func TestInt(t *testing.T) {
	tests := []struct {
		value      string
		shoudlPass bool
	}{
		{value: "1", shoudlPass: true},
		{value: "0", shoudlPass: true},
		{value: "-1", shoudlPass: true},
		{value: "", shoudlPass: false},
		{value: "aoeu", shoudlPass: false},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			testInt(t, test.value, !test.shoudlPass)
		})
	}
}

func TestEmail(t *testing.T) {
	tests := []struct {
		value      string
		expected   string
		shouldPass bool
	}{
		{value: "aoeu", shouldPass: false},
		{value: "aoeu@aoeu", shouldPass: false},
		{value: "AOEU@aoeu.com ", expected: "aoeu@aoeu.com", shouldPass: true},
		{value: "aoeuaoeuaoeuaoeuaoeuaoeu@aoeu.com ", shouldPass: false},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			testEmail(t, test.value, test.expected, !test.shouldPass)
		})
	}
}

func TestCoerceToType(t *testing.T) {
	require.Equal(t, 1, coerceToType("1").(int))
	require.Equal(t, 1.1, coerceToType("1.1").(float64))
	require.Equal(t, true, coerceToType("true").(bool))
	require.Equal(t, false, coerceToType("false").(bool))
	require.Equal(t, "a b", coerceToType("a+b").(string))
	require.Equal(t, ".1", coerceToType(".1").(string))
}

func TestFloat(t *testing.T) {
	tests := []struct {
		value      string
		shouldPass bool
	}{
		{value: "0.1", shouldPass: true},
		{value: "1.4", shouldPass: true},
		{value: "0", shouldPass: true},
		{value: "-1", shouldPass: true},
		{value: "-0.8", shouldPass: true},
		{value: ".8", shouldPass: false},
		{value: "a", shouldPass: false},
		{value: "4.9a", shouldPass: false},
		{value: "a4.9", shouldPass: false},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			testFloat(t, test.value, !test.shouldPass)
		})
	}
}

func testEmail(t *testing.T, testValue string, expectedValue string, requiresError bool) {
	k := &struct {
		Value string `required:"true" validate:"email" trimlower:"true" truncate:"15"`
	}{}

	runReqTests(t, k, testValue, requiresError, true)
	if !requiresError {
		require.Equal(t, expectedValue, k.Value, fmt.Sprintf("Email: %s", testValue))
	}
}

func TestMaxLength(t *testing.T) {
	k := &struct {
		Value string `required:"true" max-length:"5"`
	}{}

	runReqTests(t, k, "aoeu-aoeu", true, false)
}

func mustRequest(t *testing.T, value interface{}, useQuotes bool) *http.Request {
	theJSON := []byte(`{"value":` + value.(string) + `}`)
	if useQuotes {
		theJSON = []byte(`{"value":"` + value.(string) + `"}`)
	}
	request, err := http.NewRequest("GET", "/?value="+value.(string), io.NopCloser(bytes.NewReader(theJSON)))
	require.NoError(t, err)
	return request
}

func testFloat(t *testing.T, testValue string, requiresError bool) {
	k := &struct {
		Value *float64 `required:"true"`
	}{}

	runReqTests(t, k, testValue, requiresError, false)
	if !requiresError {
		floatValue, _ := strconv.ParseFloat(testValue, 64)
		require.Equal(t, floatValue, *k.Value, fmt.Sprintf("Float: %s", testValue))
	}
}

func runReqTests(t *testing.T, k interface{}, testValue interface{}, requiresError bool, useQuotes bool) {
	if requiresError {
		require.Error(t, UnmarshalQuery(mustRequest(t, testValue, useQuotes), k), fmt.Sprintf("QueryToStruct: %s", testValue))
		require.Error(t, UnmarshalBody(mustRequest(t, testValue, useQuotes), k), fmt.Sprintf("BodyToStruct: %s", testValue))
	} else {
		require.NoError(t, UnmarshalQuery(mustRequest(t, testValue, useQuotes), k), fmt.Sprintf("QueryToStruct: %s", testValue))
		require.NoError(t, UnmarshalBody(mustRequest(t, testValue, useQuotes), k), fmt.Sprintf("BodyToStruct: %s", testValue))
	}
}

func testInt(t *testing.T, testValue string, requiresError bool) {
	k := &struct {
		Value *int `required:"true"`
	}{}

	runReqTests(t, k, testValue, requiresError, false)
	if !requiresError {
		floatValue, _ := strconv.Atoi(testValue)
		require.Equal(t, floatValue, *k.Value, fmt.Sprintf("Int: %s", testValue))
	}
}
