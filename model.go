package main

type ScheduleRequestInput struct {
	Dose          int      `json:"dose"`
	SessionID     string   `json:"session_id"`
	Slot          string   `json:"slot"`
	Beneficiaries []string `json:"beneficiaries"`
}

type GetBeneficiariesResponse struct {
	Beneficiaries []Beneficiaries `json:"beneficiaries"`
}
type Appointments struct {
	CenterID      int     `json:"center_id"`
	Name          string  `json:"name"`
	NameL         string  `json:"name_l"`
	Address       string  `json:"address"`
	AddressL      string  `json:"address_l"`
	StateName     string  `json:"state_name"`
	StateNameL    string  `json:"state_name_l"`
	DistrictName  string  `json:"district_name"`
	DistrictNameL string  `json:"district_name_l"`
	BlockName     string  `json:"block_name"`
	BlockNameL    string  `json:"block_name_l"`
	Pincode       string  `json:"pincode"`
	Lat           float64 `json:"lat"`
	Long          float64 `json:"long"`
	From          string  `json:"from"`
	To            string  `json:"to"`
	FeeType       string  `json:"fee_type"`
	Dose          int     `json:"dose"`
	AppointmentID string  `json:"appointment_id"`
	SessionID     string  `json:"session_id"`
	Date          string  `json:"date"`
	Slot          string  `json:"slot"`
}
type Beneficiaries struct {
	BeneficiaryReferenceID string         `json:"beneficiary_reference_id"`
	Name                   string         `json:"name"`
	BirthYear              string         `json:"birth_year"`
	Gender                 string         `json:"gender"`
	MobileNumber           string         `json:"mobile_number"`
	PhotoIDType            string         `json:"photo_id_type"`
	PhotoIDNumber          string         `json:"photo_id_number"`
	ComorbidityInd         string         `json:"comorbidity_ind"`
	VaccinationStatus      string         `json:"vaccination_status"`
	Vaccine                string         `json:"vaccine"`
	Dose1Date              string         `json:"dose1_date"`
	Dose2Date              string         `json:"dose2_date"`
	Appointments           []Appointments `json:"appointments"`
}

type GetResponse struct {
	Centers []Centers `json:"centers"`
}
type Sessions struct {
	SessionID         string   `json:"session_id"`
	Date              string   `json:"date"`
	AvailableCapacity int      `json:"available_capacity"`
	MinAgeLimit       int      `json:"min_age_limit"`
	Vaccine           string   `json:"vaccine"`
	Slots             []string `json:"slots"`
}
type Centers struct {
	CenterID     int        `json:"center_id"`
	Name         string     `json:"name"`
	Address      string     `json:"address"`
	StateName    string     `json:"state_name"`
	DistrictName string     `json:"district_name"`
	BlockName    string     `json:"block_name"`
	Pincode      int        `json:"pincode"`
	Lat          int        `json:"lat"`
	Long         int        `json:"long"`
	From         string     `json:"from"`
	To           string     `json:"to"`
	FeeType      string     `json:"fee_type"`
	Sessions     []Sessions `json:"sessions"`
}
