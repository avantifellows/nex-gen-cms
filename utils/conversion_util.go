package utils

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"regexp"
	"strconv"
	"strings"
)

func StringToInt(s string) int {
	// Convert string to integer using strconv.Atoi
	num, err := strconv.Atoi(s)
	if err != nil {
		fmt.Println("Error:", err)
		return 0 // Return zero
	}
	return num // Return the converted integer
}

type IntType interface {
	int8 | int16 | int32
}

// Generic function to convert string to int8 or int16
func StringToIntType[T IntType](str string) (T, error) {
	// Parse the string as an int64
	num, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		fmt.Println("Error:", err)
		return 0, err
	}

	// Convert to the desired type
	var result T
	switch any(result).(type) {
	case int8:
		if num < -128 || num > 127 {
			fmt.Println("value out of range for int8")
			return 0, fmt.Errorf("value out of range for int8")
		}
	case int16:
		if num < -32768 || num > 32767 {
			fmt.Println("value out of range for int16")
			return 0, fmt.Errorf("value out of range for int16")
		}
	case int32:
		if num < -2147483648 || num > 2147483647 {
			fmt.Println("value out of range for int32")
			return 0, fmt.Errorf("value out of range for int32")
		}
	}

	result = T(num)
	return result, nil
}

func ExtractNumericSuffix(s string) int {
	// Define a regular expression to find the numeric suffix
	re := regexp.MustCompile(`[0-9]+$`)
	match := re.FindString(s)

	// Convert the matched string to an integer
	if match != "" {
		num := StringToInt(match)
		return num
	}
	// return 0 if no numeric suffix is found
	return 0
}

func JoinInt16(intArr []int16, separator string) string {
	var stringArr []string
	for _, integer := range intArr {
		stringArr = append(stringArr, strconv.Itoa(int(integer)))
	}
	return strings.Join(stringArr, separator)
}

/**
 * Custom Slice() is defined to handle any number of arguments; otherwise default Slice() has
 * restriction on number of arguments (mostly 7)
 */
func Slice(args ...any) []any {
	return args
}

func EmptySlice[T any]() []T {
	return []T{}
}

func Dict(values ...any) map[string]any {
	dict := make(map[string]any)
	for i := 0; i < len(values); i += 2 {
		key := values[i].(string)
		value := values[i+1]
		dict[key] = value
	}
	return dict
}

func DisplaySubtype(subtype string) string {
	switch subtype {
	case "mcq_single_answer":
		return "MCQ Single Answer"
	case "numerical_answer":
		return "Numerical Answer"
	case "integer_type":
		return "Integer Type"
	default:
		return "Unknown"
	}
}

func ToJson(v any) template.JS {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("Error marshalling to JSON: %v", err)
		return template.JS("[]") // fallback to empty JSON array
	}
	return template.JS(b)
}

func IntToString[T IntType](v T) string {
	return strconv.FormatInt(int64(v), 10)
}

func FloatPtr(v float64) *float64 {
	return &v
}

func Capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}

func Append[T any](slice []T, v T) []T {
	return append(slice, v)
}
