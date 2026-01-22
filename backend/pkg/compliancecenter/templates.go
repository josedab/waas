package compliancecenter

import (
	"time"

	"github.com/google/uuid"
)

// GetBuiltInTemplates returns all built-in compliance templates
func GetBuiltInTemplates() []ComplianceTemplate {
	return []ComplianceTemplate{
		SOC2Template(),
		HIPAATemplate(),
		GDPRTemplate(),
		PCIDSSTemplate(),
	}
}

// SOC2Template returns the SOC2 Type II compliance template
func SOC2Template() ComplianceTemplate {
	now := time.Now()
	return ComplianceTemplate{
		ID:          "soc2-template",
		Framework:   FrameworkSOC2,
		Name:        "SOC 2 Type II",
		Description: "Service Organization Control 2 Type II compliance framework",
		Version:     "2017",
		Controls: []Control{
			// Security (Common Criteria)
			{
				ID: "cc1", Code: "CC1.1", Name: "Control Environment",
				Description: "The entity demonstrates a commitment to integrity and ethical values",
				Category: CategoryAccessControl, Priority: "high",
				Checks: []ControlCheck{
					{ID: "cc1-1", Name: "Code of Conduct", CheckType: "manual", Description: "Verify code of conduct exists and is communicated"},
				},
			},
			{
				ID: "cc6", Code: "CC6.1", Name: "Logical and Physical Access Controls",
				Description: "The entity implements logical access security software, infrastructure, and architectures",
				Category: CategoryAccessControl, Priority: "critical",
				Checks: []ControlCheck{
					{ID: "cc6-1", Name: "Access Control Enforcement", CheckType: "automated", Description: "Verify role-based access control is enforced", Query: "check_rbac_enabled"},
					{ID: "cc6-2", Name: "Authentication Requirements", CheckType: "automated", Description: "Verify MFA is required for privileged access", Query: "check_mfa_enabled"},
				},
			},
			{
				ID: "cc6-6", Code: "CC6.6", Name: "System Boundary Protection",
				Description: "The entity implements logical access security measures to protect against threats from sources outside its system boundaries",
				Category: CategoryNetwork, Priority: "critical",
				Checks: []ControlCheck{
					{ID: "cc6-6-1", Name: "Firewall Configuration", CheckType: "automated", Description: "Verify firewall rules are properly configured"},
					{ID: "cc6-6-2", Name: "Network Segmentation", CheckType: "automated", Description: "Verify network segmentation is implemented"},
				},
			},
			{
				ID: "cc6-7", Code: "CC6.7", Name: "Transmission Security",
				Description: "The entity restricts the transmission, movement, and removal of information to authorized internal and external users and processes",
				Category: CategoryEncryption, Priority: "critical",
				Checks: []ControlCheck{
					{ID: "cc6-7-1", Name: "TLS Enforcement", CheckType: "automated", Description: "Verify TLS 1.2+ is enforced for all transmissions", Query: "check_tls_version"},
					{ID: "cc6-7-2", Name: "Data Classification", CheckType: "manual", Description: "Verify data classification scheme exists"},
				},
			},
			{
				ID: "cc7", Code: "CC7.2", Name: "System Monitoring",
				Description: "The entity monitors system components and the operation of those components for anomalies",
				Category: CategoryLogging, Priority: "high",
				Checks: []ControlCheck{
					{ID: "cc7-1", Name: "Audit Logging", CheckType: "automated", Description: "Verify comprehensive audit logging is enabled", Query: "check_audit_logging"},
					{ID: "cc7-2", Name: "Anomaly Detection", CheckType: "automated", Description: "Verify anomaly detection is configured"},
				},
			},
			{
				ID: "cc7-3", Code: "CC7.3", Name: "Security Event Evaluation",
				Description: "The entity evaluates security events to determine whether they could or have resulted in a failure",
				Category: CategoryIncidentResponse, Priority: "high",
				Checks: []ControlCheck{
					{ID: "cc7-3-1", Name: "Incident Response Plan", CheckType: "manual", Description: "Verify incident response plan exists"},
					{ID: "cc7-3-2", Name: "Event Correlation", CheckType: "automated", Description: "Verify security event correlation is enabled"},
				},
			},
			{
				ID: "cc8", Code: "CC8.1", Name: "Change Management",
				Description: "The entity authorizes, designs, develops or acquires, configures, documents, tests, approves, and implements changes",
				Category: CategoryRiskManagement, Priority: "high",
				Checks: []ControlCheck{
					{ID: "cc8-1", Name: "Change Control Process", CheckType: "manual", Description: "Verify change management process exists"},
					{ID: "cc8-2", Name: "Version Control", CheckType: "automated", Description: "Verify all code changes go through version control"},
				},
			},
			{
				ID: "cc9", Code: "CC9.1", Name: "Risk Mitigation",
				Description: "The entity identifies, selects, and develops risk mitigation activities for risks arising from potential business disruptions",
				Category: CategoryBusinessContinuity, Priority: "high",
				Checks: []ControlCheck{
					{ID: "cc9-1", Name: "Business Continuity Plan", CheckType: "manual", Description: "Verify BCP exists and is tested"},
					{ID: "cc9-2", Name: "Disaster Recovery", CheckType: "automated", Description: "Verify DR procedures are documented and tested"},
				},
			},
		},
		Policies: []PolicyTemplate{
			{
				ID: "soc2-access-policy", Name: "Access Control Policy",
				Description: "Enforces SOC2 access control requirements",
				ControlIDs: []string{"cc6", "cc6-6"},
				DefaultMode: EnforcementEnforce,
				Rules: []PolicyRule{
					{ID: "r1", Name: "Require HTTPS", Condition: "request.protocol == 'https'", Action: "deny", Severity: "critical", Message: "HTTPS is required for all connections"},
					{ID: "r2", Name: "Require Authentication", Condition: "request.authenticated == true", Action: "deny", Severity: "critical", Message: "Authentication is required"},
				},
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// HIPAATemplate returns the HIPAA compliance template
func HIPAATemplate() ComplianceTemplate {
	now := time.Now()
	return ComplianceTemplate{
		ID:          "hipaa-template",
		Framework:   FrameworkHIPAA,
		Name:        "HIPAA Security Rule",
		Description: "Health Insurance Portability and Accountability Act Security Rule",
		Version:     "2013",
		Controls: []Control{
			// Administrative Safeguards
			{
				ID: "164.308.a.1", Code: "164.308(a)(1)", Name: "Security Management Process",
				Description: "Implement policies and procedures to prevent, detect, contain, and correct security violations",
				Category: CategoryRiskManagement, Priority: "critical",
				Checks: []ControlCheck{
					{ID: "h1-1", Name: "Risk Analysis", CheckType: "manual", Description: "Conduct accurate and thorough risk analysis"},
					{ID: "h1-2", Name: "Risk Management", CheckType: "manual", Description: "Implement security measures sufficient to reduce risks"},
				},
			},
			{
				ID: "164.308.a.3", Code: "164.308(a)(3)", Name: "Workforce Security",
				Description: "Implement policies and procedures to ensure appropriate access to ePHI by workforce members",
				Category: CategoryAccessControl, Priority: "critical",
				Checks: []ControlCheck{
					{ID: "h3-1", Name: "Authorization", CheckType: "automated", Description: "Implement procedures for access authorization", Query: "check_access_authorization"},
					{ID: "h3-2", Name: "Termination Procedures", CheckType: "manual", Description: "Implement termination procedures"},
				},
			},
			{
				ID: "164.308.a.4", Code: "164.308(a)(4)", Name: "Information Access Management",
				Description: "Implement policies and procedures for authorizing access to ePHI",
				Category: CategoryAccessControl, Priority: "critical",
				Checks: []ControlCheck{
					{ID: "h4-1", Name: "Access Establishment", CheckType: "manual", Description: "Implement policies for access establishment and modification"},
				},
			},
			{
				ID: "164.308.a.5", Code: "164.308(a)(5)", Name: "Security Awareness Training",
				Description: "Implement security awareness and training program for all workforce members",
				Category: CategoryRiskManagement, Priority: "high",
				Checks: []ControlCheck{
					{ID: "h5-1", Name: "Security Training", CheckType: "manual", Description: "Provide security reminders and training"},
				},
			},
			// Technical Safeguards
			{
				ID: "164.312.a.1", Code: "164.312(a)(1)", Name: "Access Control",
				Description: "Implement technical policies and procedures for electronic information systems that maintain ePHI",
				Category: CategoryAccessControl, Priority: "critical",
				Checks: []ControlCheck{
					{ID: "ht1-1", Name: "Unique User ID", CheckType: "automated", Description: "Assign unique name/number for identifying users", Query: "check_unique_user_ids"},
					{ID: "ht1-2", Name: "Automatic Logoff", CheckType: "automated", Description: "Implement automatic logoff", Query: "check_session_timeout"},
					{ID: "ht1-3", Name: "Encryption", CheckType: "automated", Description: "Implement encryption/decryption mechanism", Query: "check_encryption"},
				},
			},
			{
				ID: "164.312.b", Code: "164.312(b)", Name: "Audit Controls",
				Description: "Implement hardware, software, and/or procedural mechanisms that record and examine activity",
				Category: CategoryLogging, Priority: "critical",
				Checks: []ControlCheck{
					{ID: "ht2-1", Name: "Audit Logging", CheckType: "automated", Description: "Implement audit controls", Query: "check_phi_audit_logging"},
				},
			},
			{
				ID: "164.312.c.1", Code: "164.312(c)(1)", Name: "Integrity Controls",
				Description: "Implement policies and procedures to protect ePHI from improper alteration or destruction",
				Category: CategoryDataProtection, Priority: "critical",
				Checks: []ControlCheck{
					{ID: "ht3-1", Name: "Integrity Mechanism", CheckType: "automated", Description: "Implement electronic mechanisms to corroborate data integrity"},
				},
			},
			{
				ID: "164.312.e.1", Code: "164.312(e)(1)", Name: "Transmission Security",
				Description: "Implement technical security measures to guard against unauthorized access to ePHI transmitted over network",
				Category: CategoryEncryption, Priority: "critical",
				Checks: []ControlCheck{
					{ID: "ht4-1", Name: "Transmission Encryption", CheckType: "automated", Description: "Implement encryption for ePHI transmission", Query: "check_tls_version"},
				},
			},
		},
		Policies: []PolicyTemplate{
			{
				ID: "hipaa-phi-policy", Name: "PHI Protection Policy",
				Description: "Enforces HIPAA PHI protection requirements",
				ControlIDs: []string{"164.312.a.1", "164.312.e.1"},
				DefaultMode: EnforcementEnforce,
				Rules: []PolicyRule{
					{ID: "r1", Name: "Encrypt PHI at Rest", Condition: "data.contains_phi && !data.encrypted", Action: "deny", Severity: "critical", Message: "PHI must be encrypted at rest"},
					{ID: "r2", Name: "Encrypt PHI in Transit", Condition: "data.contains_phi && request.protocol != 'https'", Action: "deny", Severity: "critical", Message: "PHI must be encrypted in transit"},
					{ID: "r3", Name: "Audit PHI Access", Condition: "data.contains_phi", Action: "log", Severity: "high", Message: "All PHI access must be logged"},
				},
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// GDPRTemplate returns the GDPR compliance template
func GDPRTemplate() ComplianceTemplate {
	now := time.Now()
	return ComplianceTemplate{
		ID:          "gdpr-template",
		Framework:   FrameworkGDPR,
		Name:        "GDPR",
		Description: "General Data Protection Regulation (EU) 2016/679",
		Version:     "2018",
		Controls: []Control{
			{
				ID: "art5", Code: "Article 5", Name: "Data Processing Principles",
				Description: "Personal data shall be processed lawfully, fairly and in a transparent manner",
				Category: CategoryDataProtection, Priority: "critical",
				Checks: []ControlCheck{
					{ID: "g5-1", Name: "Lawfulness of Processing", CheckType: "manual", Description: "Verify lawful basis for processing exists"},
					{ID: "g5-2", Name: "Purpose Limitation", CheckType: "manual", Description: "Verify data collected for specified purposes only"},
					{ID: "g5-3", Name: "Data Minimization", CheckType: "automated", Description: "Verify only necessary data is collected"},
				},
			},
			{
				ID: "art6", Code: "Article 6", Name: "Lawfulness of Processing",
				Description: "Processing shall be lawful only if and to the extent that at least one legal basis applies",
				Category: CategoryDataProtection, Priority: "critical",
				Checks: []ControlCheck{
					{ID: "g6-1", Name: "Consent Management", CheckType: "automated", Description: "Verify consent is properly obtained and recorded", Query: "check_consent_management"},
				},
			},
			{
				ID: "art17", Code: "Article 17", Name: "Right to Erasure",
				Description: "The data subject shall have the right to obtain erasure of personal data",
				Category: CategoryDataProtection, Priority: "critical",
				Checks: []ControlCheck{
					{ID: "g17-1", Name: "Data Deletion Capability", CheckType: "automated", Description: "Verify ability to delete personal data on request", Query: "check_data_deletion"},
				},
			},
			{
				ID: "art20", Code: "Article 20", Name: "Right to Data Portability",
				Description: "The data subject shall have the right to receive personal data in a structured format",
				Category: CategoryDataProtection, Priority: "high",
				Checks: []ControlCheck{
					{ID: "g20-1", Name: "Data Export Capability", CheckType: "automated", Description: "Verify ability to export personal data", Query: "check_data_export"},
				},
			},
			{
				ID: "art25", Code: "Article 25", Name: "Data Protection by Design",
				Description: "Implement appropriate technical and organizational measures designed to implement data protection principles",
				Category: CategoryDataProtection, Priority: "critical",
				Checks: []ControlCheck{
					{ID: "g25-1", Name: "Privacy by Design", CheckType: "manual", Description: "Verify privacy by design principles are implemented"},
					{ID: "g25-2", Name: "Default Privacy Settings", CheckType: "automated", Description: "Verify privacy-protective defaults are in place"},
				},
			},
			{
				ID: "art30", Code: "Article 30", Name: "Records of Processing Activities",
				Description: "Maintain a record of processing activities under responsibility of controller",
				Category: CategoryLogging, Priority: "high",
				Checks: []ControlCheck{
					{ID: "g30-1", Name: "Processing Records", CheckType: "manual", Description: "Verify records of processing activities are maintained"},
				},
			},
			{
				ID: "art32", Code: "Article 32", Name: "Security of Processing",
				Description: "Implement appropriate technical and organizational measures to ensure security",
				Category: CategoryEncryption, Priority: "critical",
				Checks: []ControlCheck{
					{ID: "g32-1", Name: "Encryption", CheckType: "automated", Description: "Verify encryption of personal data", Query: "check_encryption"},
					{ID: "g32-2", Name: "Confidentiality", CheckType: "automated", Description: "Ensure confidentiality of processing systems"},
					{ID: "g32-3", Name: "Availability", CheckType: "automated", Description: "Ensure availability and resilience of processing systems"},
				},
			},
			{
				ID: "art33", Code: "Article 33", Name: "Breach Notification",
				Description: "Notify supervisory authority of personal data breach within 72 hours",
				Category: CategoryIncidentResponse, Priority: "critical",
				Checks: []ControlCheck{
					{ID: "g33-1", Name: "Breach Detection", CheckType: "automated", Description: "Verify breach detection mechanisms are in place"},
					{ID: "g33-2", Name: "Notification Process", CheckType: "manual", Description: "Verify breach notification process exists"},
				},
			},
			{
				ID: "art44", Code: "Article 44", Name: "International Data Transfers",
				Description: "Transfer of personal data to third countries subject to conditions",
				Category: CategoryDataProtection, Priority: "critical",
				Checks: []ControlCheck{
					{ID: "g44-1", Name: "Transfer Mechanisms", CheckType: "automated", Description: "Verify appropriate transfer mechanisms in place", Query: "check_data_residency"},
				},
			},
		},
		Policies: []PolicyTemplate{
			{
				ID: "gdpr-data-policy", Name: "GDPR Data Protection Policy",
				Description: "Enforces GDPR data protection requirements",
				ControlIDs: []string{"art5", "art32", "art44"},
				DefaultMode: EnforcementEnforce,
				Rules: []PolicyRule{
					{ID: "r1", Name: "EU Data Residency", Condition: "data.subject_location == 'EU' && !data.stored_in_eu", Action: "deny", Severity: "critical", Message: "EU personal data must be stored in EU or approved country"},
					{ID: "r2", Name: "Encryption Required", Condition: "data.is_personal && !data.encrypted", Action: "deny", Severity: "high", Message: "Personal data must be encrypted"},
					{ID: "r3", Name: "Consent Required", Condition: "data.is_personal && !data.has_consent", Action: "deny", Severity: "critical", Message: "Valid consent required for personal data processing"},
				},
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// PCIDSSTemplate returns the PCI-DSS compliance template
func PCIDSSTemplate() ComplianceTemplate {
	now := time.Now()
	return ComplianceTemplate{
		ID:          "pci-dss-template",
		Framework:   FrameworkPCIDSS,
		Name:        "PCI-DSS",
		Description: "Payment Card Industry Data Security Standard",
		Version:     "4.0",
		Controls: []Control{
			// Requirement 1: Network Security Controls
			{
				ID: "req1", Code: "1.1", Name: "Network Security Controls",
				Description: "Install and maintain network security controls",
				Category: CategoryNetwork, Priority: "critical",
				Checks: []ControlCheck{
					{ID: "p1-1", Name: "Firewall Configuration", CheckType: "automated", Description: "Verify firewall configuration standards"},
					{ID: "p1-2", Name: "Network Diagram", CheckType: "manual", Description: "Verify current network diagram exists"},
				},
			},
			// Requirement 3: Protect Stored Account Data
			{
				ID: "req3", Code: "3.1", Name: "Protect Stored Data",
				Description: "Protect stored account data",
				Category: CategoryEncryption, Priority: "critical",
				Checks: []ControlCheck{
					{ID: "p3-1", Name: "Data Encryption", CheckType: "automated", Description: "Verify cardholder data is encrypted", Query: "check_card_data_encryption"},
					{ID: "p3-2", Name: "Key Management", CheckType: "manual", Description: "Verify encryption key management process"},
				},
			},
			// Requirement 4: Protect Cardholder Data in Transit
			{
				ID: "req4", Code: "4.1", Name: "Encrypt Transmission",
				Description: "Protect cardholder data with strong cryptography during transmission",
				Category: CategoryEncryption, Priority: "critical",
				Checks: []ControlCheck{
					{ID: "p4-1", Name: "TLS Configuration", CheckType: "automated", Description: "Verify TLS 1.2+ is required", Query: "check_tls_version"},
					{ID: "p4-2", Name: "Certificate Management", CheckType: "automated", Description: "Verify valid certificates are in use"},
				},
			},
			// Requirement 7: Restrict Access
			{
				ID: "req7", Code: "7.1", Name: "Restrict Access",
				Description: "Restrict access to cardholder data by business need to know",
				Category: CategoryAccessControl, Priority: "critical",
				Checks: []ControlCheck{
					{ID: "p7-1", Name: "Access Control System", CheckType: "automated", Description: "Verify access control system limits access", Query: "check_rbac_enabled"},
					{ID: "p7-2", Name: "Role-Based Access", CheckType: "automated", Description: "Verify role-based access is implemented"},
				},
			},
			// Requirement 8: Identify Users
			{
				ID: "req8", Code: "8.1", Name: "Identify Users",
				Description: "Identify users and authenticate access",
				Category: CategoryAccessControl, Priority: "critical",
				Checks: []ControlCheck{
					{ID: "p8-1", Name: "Unique User IDs", CheckType: "automated", Description: "Verify unique IDs for all users", Query: "check_unique_user_ids"},
					{ID: "p8-2", Name: "Strong Authentication", CheckType: "automated", Description: "Verify strong authentication is required", Query: "check_mfa_enabled"},
				},
			},
			// Requirement 10: Log and Monitor Access
			{
				ID: "req10", Code: "10.1", Name: "Log and Monitor",
				Description: "Log and monitor all access to system components and cardholder data",
				Category: CategoryLogging, Priority: "critical",
				Checks: []ControlCheck{
					{ID: "p10-1", Name: "Audit Trails", CheckType: "automated", Description: "Verify audit trails are enabled", Query: "check_audit_logging"},
					{ID: "p10-2", Name: "Log Retention", CheckType: "automated", Description: "Verify logs are retained for at least one year"},
				},
			},
			// Requirement 11: Test Security Systems
			{
				ID: "req11", Code: "11.1", Name: "Test Security",
				Description: "Test security of systems and networks regularly",
				Category: CategoryRiskManagement, Priority: "high",
				Checks: []ControlCheck{
					{ID: "p11-1", Name: "Vulnerability Scans", CheckType: "manual", Description: "Verify quarterly vulnerability scans are performed"},
					{ID: "p11-2", Name: "Penetration Testing", CheckType: "manual", Description: "Verify annual penetration testing is performed"},
				},
			},
			// Requirement 12: Organizational Policies
			{
				ID: "req12", Code: "12.1", Name: "Security Policies",
				Description: "Support information security with organizational policies and programs",
				Category: CategoryRiskManagement, Priority: "high",
				Checks: []ControlCheck{
					{ID: "p12-1", Name: "Security Policy", CheckType: "manual", Description: "Verify information security policy exists"},
					{ID: "p12-2", Name: "Security Awareness", CheckType: "manual", Description: "Verify security awareness program exists"},
				},
			},
		},
		Policies: []PolicyTemplate{
			{
				ID: "pci-dss-policy", Name: "PCI-DSS Data Protection Policy",
				Description: "Enforces PCI-DSS cardholder data protection requirements",
				ControlIDs: []string{"req3", "req4", "req10"},
				DefaultMode: EnforcementEnforce,
				Rules: []PolicyRule{
					{ID: "r1", Name: "Encrypt Card Data", Condition: "data.contains_pan && !data.encrypted", Action: "deny", Severity: "critical", Message: "Cardholder data must be encrypted"},
					{ID: "r2", Name: "Mask PAN", Condition: "data.contains_pan && !data.masked", Action: "deny", Severity: "critical", Message: "PAN must be masked when displayed"},
					{ID: "r3", Name: "No CVV Storage", Condition: "data.contains_cvv", Action: "deny", Severity: "critical", Message: "CVV/CVC must never be stored"},
					{ID: "r4", Name: "Log Card Data Access", Condition: "data.contains_pan", Action: "log", Severity: "high", Message: "All cardholder data access must be logged"},
				},
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// CreateDefaultPolicies creates default policies for common use cases
func CreateDefaultPolicies() []PolicyTemplate {
	return []PolicyTemplate{
		{
			ID:          uuid.New().String(),
			Name:        "Encryption Enforcement",
			Description: "Ensures all sensitive data is encrypted at rest and in transit",
			ControlIDs:  []string{},
			DefaultMode: EnforcementEnforce,
			Rules: []PolicyRule{
				{ID: "enc-1", Name: "TLS Required", Condition: "request.protocol != 'https'", Action: "deny", Severity: "critical", Message: "HTTPS/TLS is required for all connections"},
				{ID: "enc-2", Name: "Encryption at Rest", Condition: "data.sensitive && !data.encrypted", Action: "deny", Severity: "high", Message: "Sensitive data must be encrypted at rest"},
			},
		},
		{
			ID:          uuid.New().String(),
			Name:        "Access Control",
			Description: "Enforces proper authentication and authorization",
			ControlIDs:  []string{},
			DefaultMode: EnforcementEnforce,
			Rules: []PolicyRule{
				{ID: "acc-1", Name: "Authentication Required", Condition: "!request.authenticated", Action: "deny", Severity: "critical", Message: "Authentication is required"},
				{ID: "acc-2", Name: "MFA Required for Admin", Condition: "request.role == 'admin' && !request.mfa_verified", Action: "deny", Severity: "critical", Message: "MFA is required for admin access"},
			},
		},
		{
			ID:          uuid.New().String(),
			Name:        "Audit Logging",
			Description: "Ensures all security-relevant events are logged",
			ControlIDs:  []string{},
			DefaultMode: EnforcementAudit,
			Rules: []PolicyRule{
				{ID: "log-1", Name: "Log Admin Actions", Condition: "request.role == 'admin'", Action: "log", Severity: "high", Message: "Admin actions must be logged"},
				{ID: "log-2", Name: "Log Data Access", Condition: "data.sensitive", Action: "log", Severity: "medium", Message: "Sensitive data access must be logged"},
			},
		},
		{
			ID:          uuid.New().String(),
			Name:        "Data Residency",
			Description: "Enforces data residency requirements",
			ControlIDs:  []string{},
			DefaultMode: EnforcementEnforce,
			Rules: []PolicyRule{
				{ID: "res-1", Name: "EU Data in EU", Condition: "data.subject_region == 'EU' && request.destination_region not in ['EU', 'EEA']", Action: "deny", Severity: "critical", Message: "EU data must be processed in EU/EEA regions"},
			},
		},
	}
}
