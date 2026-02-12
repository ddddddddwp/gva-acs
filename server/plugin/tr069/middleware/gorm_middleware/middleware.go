package gormmiddleware

import (
	"reflect"
	"strings"

	"gorm.io/gorm"
)

type Rule struct {
	Table        string
	Field        string
	DenyPrefixes []string
}

type Middleware struct {
	rules   []Rule
	byTable map[string][]Rule
}

func New(rules []Rule) *Middleware {
	cp := make([]Rule, 0, len(rules))
	for _, r := range rules {
		cp = append(cp, r)
	}
	return &Middleware{rules: cp}
}

func (m *Middleware) Name() string {
	return "gormingestfilter"
}

func (m *Middleware) Initialize(db *gorm.DB) error {
	m.byTable = make(map[string][]Rule, len(m.rules))
	for _, r := range m.rules {
		if r.Table == "" || r.Field == "" || len(r.DenyPrefixes) == 0 {
			continue
		}
		key := normalizeTableName(r.Table)
		if key == "" {
			continue
		}
		m.byTable[key] = append(m.byTable[key], r)
	}

	return db.Callback().Create().Before("gorm:create").Register("gormingestfilter:before_create", func(tx *gorm.DB) {
		if tx == nil || tx.Statement == nil || tx.Statement.Dest == nil {
			return
		}

		table := tx.Statement.Table
		if table == "" && tx.Statement.Schema != nil {
			table = tx.Statement.Schema.Table
		}
		table = normalizeTableName(table)
		if table == "" {
			return
		}

		rules := m.byTable[table]
		if len(rules) == 0 {
			return
		}

		newDest, replaced := filterDest(tx.Statement.Dest, rules)
		if replaced {
			tx.Statement.Dest = newDest
			tx.Statement.ReflectValue = reflect.ValueOf(newDest)
			if tx.Statement.ReflectValue.IsValid() && tx.Statement.ReflectValue.Kind() == reflect.Ptr {
				if !tx.Statement.ReflectValue.IsNil() {
					tx.Statement.ReflectValue = tx.Statement.ReflectValue.Elem()
				}
			}
		}
	})
}
func filterDest(dest any, rules []Rule) (any, bool) {
	v := reflect.ValueOf(dest)

	if !v.IsValid() {
		return dest, false
	}

	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return dest, false
		}
		elem := v.Elem()
		switch elem.Kind() {
		case reflect.Struct:
			if shouldDenyStruct(elem, rules) {
				return emptySliceDest(elem.Type()), true
			}
			return dest, false
		case reflect.Slice:
			filtered := filterSlice(elem, rules)
			elem.Set(filtered)
			return dest, false
		default:
			return dest, false
		}
	case reflect.Struct:
		if shouldDenyStruct(v, rules) {
			return emptySliceDest(v.Type()), true
		}
		return dest, false
	case reflect.Slice:
		filtered := filterSlice(v, rules)
		return filtered.Interface(), true
	default:
		return dest, false
	}
}

func filterSlice(slice reflect.Value, rules []Rule) reflect.Value {
	if slice.Kind() != reflect.Slice {
		return slice
	}
	out := reflect.MakeSlice(slice.Type(), 0, slice.Len())
	for i := 0; i < slice.Len(); i++ {
		item := slice.Index(i)
		if !item.IsValid() {
			continue
		}
		if item.Kind() == reflect.Interface {
			if item.IsNil() {
				out = reflect.Append(out, item)
				continue
			}
			item = item.Elem()
		}
		if item.Kind() == reflect.Ptr {
			if item.IsNil() {
				out = reflect.Append(out, item)
				continue
			}
			if item.Elem().Kind() == reflect.Struct && shouldDenyStruct(item.Elem(), rules) {
				continue
			}
			out = reflect.Append(out, item)
			continue
		}
		if item.Kind() == reflect.Struct && shouldDenyStruct(item, rules) {
			continue
		}
		out = reflect.Append(out, item)
	}
	return out
}

func emptySliceDest(elemType reflect.Type) any {
	return reflect.MakeSlice(reflect.SliceOf(elemType), 0, 0).Interface()
}

func shouldDenyStruct(s reflect.Value, rules []Rule) bool {
	if s.Kind() != reflect.Struct {
		return false
	}
	for _, r := range rules {
		name, ok := readStringField(s, r.Field)
		if !ok {
			continue
		}
		if matchAnyPrefix(name, r.DenyPrefixes) {
			return true
		}
	}
	return false
}

func readStringField(s reflect.Value, field string) (string, bool) {
	if s.Kind() != reflect.Struct {
		return "", false
	}
	f := s.FieldByName(field)
	if !f.IsValid() {
		return "", false
	}
	switch f.Kind() {
	case reflect.String:
		return f.String(), true
	case reflect.Ptr:
		if f.IsNil() {
			return "", true
		}
		if f.Elem().Kind() == reflect.String {
			return f.Elem().String(), true
		}
		return "", false
	default:
		return "", false
	}
}

func matchAnyPrefix(s string, prefixes []string) bool {
	if s == "" || len(prefixes) == 0 {
		return false
	}
	for _, p := range prefixes {
		if p == "" {
			continue
		}
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func DenyByPrefixes(s string, prefixes []string) bool {
	return matchAnyPrefix(s, prefixes)
}

func FilterDest(dest any, rules []Rule) (any, bool) {
	return filterDest(dest, rules)
}

func FilterForTable(dest any, table string, rules []Rule) (any, bool) {
	t := normalizeTableName(table)
	if t == "" || len(rules) == 0 {
		return dest, false
	}
	selected := make([]Rule, 0, len(rules))
	for _, r := range rules {
		if normalizeTableName(r.Table) == t {
			selected = append(selected, r)
		}
	}
	if len(selected) == 0 {
		return dest, false
	}
	return filterDest(dest, selected)
}

func RulesForPrefixDeny(table string, field string, prefixes []string) []Rule {
	if table == "" || field == "" || len(prefixes) == 0 {
		return nil
	}
	return []Rule{{Table: table, Field: field, DenyPrefixes: prefixes}}
}

func normalizeTableName(table string) string {
	table = strings.TrimSpace(table)
	if table == "" {
		return ""
	}
	table = strings.Trim(table, "`\"")
	if i := strings.LastIndex(table, "."); i >= 0 {
		table = table[i+1:]
	}
	table = strings.Trim(table, "`\"")
	return table
}
