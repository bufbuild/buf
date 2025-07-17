package bufanalysis

// CodeQualityReport represents a list of code quality violations.
type CodeQualityReport []CodeQualityViolation

// CodeQualityViolation describes a single violation in GitLab's code quality format.
type CodeQualityViolation struct {
	Description string              `json:"description"` // Human-readable message
	CheckName   string              `json:"check_name"`  // Unique name of the rule or check
	Fingerprint string              `json:"fingerprint"` // Unique identifier for the violation
	Location    CodeQualityLocation `json:"location"`    // Location in source
	Severity    string              `json:"severity"`    // One of: info, minor, major, critical, blocker
}

// CodeQualityLocation indicates where in the file the violation occurred.
type CodeQualityLocation struct {
	Path      string                    `json:"path"`                // Relative file path
	Lines     *CodeQualityLineRange     `json:"lines,omitempty"`     // Optional line-based location
	Positions *CodeQualityPositionRange `json:"positions,omitempty"` // Optional position-based location
}

// CodeQualityLineRange is used when only the line number is known.
type CodeQualityLineRange struct {
	Begin int `json:"begin"` // Line number where the issue starts
}

// CodeQualityPositionRange provides more precise location info.
type CodeQualityPositionRange struct {
	Begin CodeQualityPosition `json:"begin"` // Position (line/column)
}

// CodeQualityPosition identifies a character position in the file.
type CodeQualityPosition struct {
	Line int `json:"line"` // Line number (1-based)
}
