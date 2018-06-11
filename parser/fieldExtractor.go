package parser

import (
	"fmt"
	"go/ast"
	"log"

	"github.com/MarcGrol/golangAnnotations/model"
)

func extractFieldList(fieldList *ast.FieldList, imports map[string]string) []model.Field {
	mFields := make([]model.Field, 0)
	if fieldList != nil {
		for _, field := range fieldList.List {
			mFields = append(mFields, extractFields(field, imports)...)
		}
	}
	return mFields
}

func extractFields(field *ast.Field, imports map[string]string) []model.Field {
	mFields := make([]model.Field, 0)
	if field != nil {
		if len(field.Names) == 0 {
			if f, ok := extractField(field, imports); ok {
				mFields = append(mFields, f)
			}
		} else {
			// A single field can refer to multiple: example: x,y int -> x int, y int
			for _, name := range field.Names {
				if field, ok := extractField(field, imports); ok {
					field.Name = name.Name
					mFields = append(mFields, field)
				}
			}
		}
	}
	return mFields
}

func extractField(field *ast.Field, imports map[string]string) (model.Field, bool) {
	mField := model.Field{
		DocLines:     extractComments(field.Doc),
		CommentLines: extractComments(field.Comment),
		Tag:          extractTag(field.Tag),
	}

	if extractSliceField(field.Type, &mField, imports) {
		return mField, true
	}

	if extractMapField(field.Type, &mField, imports) {
		return mField, true
	}

	if extractPointerField(field.Type, &mField, imports) {
		return mField, true
	}

	if extractIdentField(field.Type, &mField, imports) {
		return mField, true
	}

	if extractSelectorField(field.Type, &mField, imports) {
		return mField, true
	}

	log.Printf("*** Could not understand field %+v: '%+v (%s)'", field, field.Names, field.Type)

	return mField, false
}

func extractSliceField(fieldType ast.Expr, mField *model.Field, imports map[string]string) bool {
	if arrayType, ok := fieldType.(*ast.ArrayType); ok {
		mField.IsSlice = true
		if extractSliceSelectorField(arrayType, mField, imports) {
			return true
		}
		if extractSlicePointerField(arrayType, mField, imports) {
			return true
		}
	}
	return false
}

func extractSliceSelectorField(arrayType *ast.ArrayType, mField *model.Field, imports map[string]string) bool {
	if ident, ok := arrayType.Elt.(*ast.Ident); ok {
		mField.TypeName = ident.Name
		return true
	}

	if selectorExpr, ok := arrayType.Elt.(*ast.SelectorExpr); ok {
		if ident, ok := selectorExpr.X.(*ast.Ident); ok {
			mField.TypeName = fmt.Sprintf("%s.%s", ident.Name, selectorExpr.Sel.Name)
			mField.PackageName = imports[ident.Name]
			return true
		}
	}
	return false
}

func extractSlicePointerField(arrayType *ast.ArrayType, mField *model.Field, imports map[string]string) bool {
	if starExpr, ok := arrayType.Elt.(*ast.StarExpr); ok {
		if ident, ok := starExpr.X.(*ast.Ident); ok {
			mField.TypeName = ident.Name
			mField.IsPointer = true
			return true
		}

		if selectorExpr, ok := starExpr.X.(*ast.SelectorExpr); ok {
			if ident, ok := selectorExpr.X.(*ast.Ident); ok {
				mField.PackageName = imports[ident.Name]
				mField.IsPointer = true
				mField.TypeName = fmt.Sprintf("%s.%s", ident.Name, selectorExpr.Sel.Name)
				return true
			}
		}
	}
	return false
}

func extractMapField(fieldType ast.Expr, mField *model.Field, imports map[string]string) bool {
	mapKey := ""
	mapValue := ""

	if mapType, ok := fieldType.(*ast.MapType); ok {
		if key, ok := mapType.Key.(*ast.Ident); ok {
			mapKey = key.Name
		}

		if value, ok := mapType.Value.(*ast.Ident); ok {
			mapValue = value.Name
		}

		if value, ok := mapType.Value.(*ast.StarExpr); ok {
			if ident, ok := value.X.(*ast.Ident); ok {
				mapValue = fmt.Sprintf("*%s", ident.Name)
			}
		}

		if value, ok := mapType.Value.(*ast.ArrayType); ok {

			if ident, ok := value.Elt.(*ast.Ident); ok {
				mapValue = fmt.Sprintf("[]%s", ident.Name)
			}

			if selectorExpr, ok := value.Elt.(*ast.SelectorExpr); ok {
				if ident, ok := selectorExpr.X.(*ast.Ident); ok {
					mapValue = fmt.Sprintf("%s.%s", ident.Name, selectorExpr.Sel.Name)
				}
			}

			if starExpr, ok := value.Elt.(*ast.StarExpr); ok {
				if ident, ok := starExpr.X.(*ast.Ident); ok {
					mapValue = fmt.Sprintf("[]*%s", ident.Name)
				}
			}
		}
	}

	if mapKey != "" && mapValue != "" {
		mField.TypeName = fmt.Sprintf("map[%s]%s", mapKey, mapValue)
		mField.IsMap = true
		return true
	}

	return false
}

func extractPointerField(fieldType ast.Expr, mField *model.Field, imports map[string]string) bool {
	{
		if starExpr, ok := fieldType.(*ast.StarExpr); ok {
			if ident, ok := starExpr.X.(*ast.Ident); ok {
				mField.TypeName = ident.Name
				mField.IsPointer = true
				return true
			}

			if selectorExpr, ok := starExpr.X.(*ast.SelectorExpr); ok {
				if ident, ok := selectorExpr.X.(*ast.Ident); ok {
					mField.TypeName = fmt.Sprintf("%s.%s", ident.Name, selectorExpr.Sel.Name)
					mField.IsPointer = true
					mField.PackageName = imports[ident.Name]
					return true
				}
			}
		}
	}
	return false
}

func extractIdentField(fieldType ast.Expr, mField *model.Field, imports map[string]string) bool {
	if ident, ok := fieldType.(*ast.Ident); ok {
		mField.TypeName = ident.Name
		return true
	}
	return false
}

func extractSelectorField(fieldType ast.Expr, mField *model.Field, imports map[string]string) bool {
	if selectorExpr, ok := fieldType.(*ast.SelectorExpr); ok {
		if ident, ok := selectorExpr.X.(*ast.Ident); ok {
			mField.Name = ident.Name
			mField.TypeName = fmt.Sprintf("%s.%s", ident.Name, selectorExpr.Sel.Name)
			mField.PackageName = imports[ident.Name]
			return true
		}
	}
	return false
}