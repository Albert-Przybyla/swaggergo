package gin

import (
	"reflect"
	"strings"
)

func extractTagValue(rawTag string, key string) (string, bool) {
	tag := reflect.StructTag(strings.Trim(rawTag, "`"))
	value, ok := tag.Lookup(key)
	if !ok {
		return "", false
	}

	head := value
	if idx := strings.Index(head, ","); idx >= 0 {
		head = head[:idx]
	}

	return head, true
}

func extractFullTagValue(rawTag string, key string) (string, bool) {
	tag := reflect.StructTag(strings.Trim(rawTag, "`"))
	return tag.Lookup(key)
}

func hasBindingOption(rawTag string, option string) bool {
	tag := reflect.StructTag(strings.Trim(rawTag, "`"))
	value, ok := tag.Lookup("binding")
	if !ok {
		return false
	}

	for _, item := range strings.Split(value, ",") {
		if strings.TrimSpace(item) == option {
			return true
		}
	}

	return false
}
