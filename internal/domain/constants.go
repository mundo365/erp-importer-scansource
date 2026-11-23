package domain

const (
	SupplierCode = "scansource"

	TermP1M     = "P1M"
	TermP1Y     = "P1Y"
	TermP3Y     = "P3Y"
	TermOneTime = "one-time"

	BillingCycleMonthly   = "Monthly"
	BillingCycleYearly    = "Yearly"
	BillingCycleTriennial = "Triennial"
	BillingCycleNone      = "None"
	BillingCycleOneTime   = "one-time"

	SegmentCorporate  = "corporate"
	SegmentAcademic   = "academic"
	SegmentCharity    = "charity"
	SegmentGovernment = "government"

	LifecycleDraft  = "draft"
	BillingSingle   = "single"
	CurrencyBRL     = "BRL"

	SheetCSP        = "CSP"
	SheetSoftSubs   = "Soft Subs"
	SheetPerpetual  = "Soft Perpétuo"

	StageParse  = "parse"
	StageUpsert = "upsert"
)
