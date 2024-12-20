package fireorm

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"reflect"
)

// SetIDField tries to set the "ID" field if it exists and is of type string.
func SetIDField(model interface{}, id string) {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	field := v.FieldByName("ID")
	if field.IsValid() && field.CanSet() && field.Kind() == reflect.String {
		field.SetString(id)
	}
}

// StructToMap converts a struct to a map (for Firestore), using the "firestore" tag for field names.
func StructToMap(model interface{}) (map[string]interface{}, error) {
	data := make(map[string]interface{})
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		fieldDef := t.Field(i)
		firestoreTag := fieldDef.Tag.Get("firestore")
		if firestoreTag == "" || firestoreTag == "-" {
			continue
		}
		fieldVal := v.Field(i)
		data[firestoreTag] = fieldVal.Interface()
	}
	return data, nil
}

// IsNotFoundError checks if the provided error corresponds to a 'NotFound' or 'Unknown' gRPC status code.
//
// Parameters:
//   - err: The error to be checked, which is expected to be a gRPC error.
//
// Returns:
//   - bool: Returns true if the error has a gRPC status code of 'NotFound' or 'Unknown', otherwise false.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	statusCode := status.Code(err)
	return statusCode == codes.NotFound || statusCode == codes.Unknown
}
