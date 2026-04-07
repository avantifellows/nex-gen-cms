package views

func GetSectionName(defaultType string, customName string) string {
	if customName != "" {
		return customName
	}

	switch defaultType {
	case "mcq_single_answer":
		return "MCQ Single Answer"
	case "mcq_multiple_answer":
		return "MCQ Multiple Answer"
	case "numerical_answer":
		return "Numerical Answer"
	case "integer_type":
		return "Integer Type"
	default:
		return "Unknown"
	}
}
