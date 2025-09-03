package submitter

import "strings"

func getSystemPrompt(executer string, target string) string {
	if executer == "submitter" {
		return "You are an expert at structured data extraction. " +
			"You will be given unstructured text from a financial report and should " +
			"convert it into the given JSON structure."
	}

	var sectionName string
	switch target {
	case "balance":
		sectionName = "Balance Sheet"
	case "income":
		sectionName = "Income Statement"
	default:
		sectionName = "Statement of Cash Flow"
	}

	return "You are a meticulous text organizer. Your job is to extract the " +
		sectionName + " section from provided text that may include messy or disorganized chunks."

}

func promptEngineerCleaner(
	contentToClean, target string, units int64) string {

	var sectionName string
	switch target {
	case "balance":
		sectionName = "Balance Sheet"
	case "income":
		sectionName = "Income Statement"
	default:
		sectionName = "Statement of Cash Flow"
	}

	var firstGuideline string

	if units == 0 {
		firstGuideline = "* Include units and time periods for the " + sectionName + ".\n"
	} else {
		firstGuideline = "* Include time periods for the " + sectionName + ".\n"
	}

	prompt := "Guidelines:\n" + firstGuideline +
		"* Ensure you capture all paragraphs that belong to the " + sectionName + ". Be careful, as the " + sectionName + " might be split into multiple paragraphs.\n" +
		"* Keep the headers and footers of the " + sectionName + " intact.\n" +
		"* Place the periods of the " + sectionName + " at the top.\n" +
		"* Do not format the text.\n\n" +
		"This is the text:\n" + contentToClean

	return prompt
}

func promptEngineerSubmitter(text, period string) string {

	year := period[:4]

	formatPeriod := func(p string) string {
		if p[len(p)-1] == 'Y' {
			return "the full " + year + " year."
		} else if p[5] == 'Q' {
			quarters := []string{"first", "second", "third", "fourth"}
			q := int(p[6] - '1')
			if q >= 0 && q < 4 {
				return "the " + quarters[q] + " quarter of " + year
			}
		} else if p[5] == 'S' {
			if p[6] == '1' {
				return "the first semester of " + year + "."
			} else {
				return "the second semester of " + year + "."
			}
		}
		return p + "." // fallback
	}

	var quarterGuideline string

	if period[5] == 'Q' {
		lowercaseText := strings.ToLower(text)
		if strings.Contains(lowercaseText, "nine months") ||
			strings.Contains(lowercaseText, "twelve months") {
			quarterGuideline = ", specifically on the three months from " + year + "."
		} else {
			quarterGuideline = "."
		}
	}

	formattedPeriod := formatPeriod(period)

	prompt := "Extract financial data from " + formattedPeriod +
		"\nGuidelines:\n" +
		"* All resulting values should be floats\n" +
		"* Do not multiply or divide values by units. Use the values as they are.\n" +
		"* Values in each row align with the years/periods as ordered in the document header or footer.\n" +
		"* For missing values that can be calculated, perform basic calculations.\n" +
		"* For values that cannot be found or calculated, use 'null'.\n" +
		"* Focus on the data from " + formattedPeriod + quarterGuideline +
		"\nThis is the text:\n" +
		text

	return prompt
}
