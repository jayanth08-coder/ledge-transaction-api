package components

import "fmt"

// FormatMoneyExport formats whole-rupee amounts for display in templates.
func FormatMoneyExport(rupees int64) string {
	return formatMoney(rupees)
}

func formatMoney(rupees int64) string {
	return fmt.Sprintf("₹%d", rupees)
}
