// Package country defines ISO 3166-1 alpha-2 country codes.
package country

// Code is an ISO 3166-1 alpha-2 country code, e.g. "MX".
type Code string

// Valid reports whether c is an officially assigned ISO 3166-1 alpha-2 code.
func (c Code) Valid() bool {
	_, ok := names[c]
	return ok
}

// Name returns the country's English short name, or "" if c isn't valid.
func (c Code) Name() string {
	return names[c]
}

// Every officially assigned ISO 3166-1 alpha-2 code.
const (
	AD Code = "AD" // Andorra
	AE Code = "AE" // United Arab Emirates
	AF Code = "AF" // Afghanistan
	AG Code = "AG" // Antigua and Barbuda
	AI Code = "AI" // Anguilla
	AL Code = "AL" // Albania
	AM Code = "AM" // Armenia
	AO Code = "AO" // Angola
	AQ Code = "AQ" // Antarctica
	AR Code = "AR" // Argentina
	AS Code = "AS" // American Samoa
	AT Code = "AT" // Austria
	AU Code = "AU" // Australia
	AW Code = "AW" // Aruba
	AX Code = "AX" // Åland Islands
	AZ Code = "AZ" // Azerbaijan
	BA Code = "BA" // Bosnia and Herzegovina
	BB Code = "BB" // Barbados
	BD Code = "BD" // Bangladesh
	BE Code = "BE" // Belgium
	BF Code = "BF" // Burkina Faso
	BG Code = "BG" // Bulgaria
	BH Code = "BH" // Bahrain
	BI Code = "BI" // Burundi
	BJ Code = "BJ" // Benin
	BL Code = "BL" // Saint Barthélemy
	BM Code = "BM" // Bermuda
	BN Code = "BN" // Brunei Darussalam
	BO Code = "BO" // Bolivia
	BQ Code = "BQ" // Bonaire, Sint Eustatius and Saba
	BR Code = "BR" // Brazil
	BS Code = "BS" // Bahamas
	BT Code = "BT" // Bhutan
	BV Code = "BV" // Bouvet Island
	BW Code = "BW" // Botswana
	BY Code = "BY" // Belarus
	BZ Code = "BZ" // Belize
	CA Code = "CA" // Canada
	CC Code = "CC" // Cocos (Keeling) Islands
	CD Code = "CD" // Democratic Republic of the Congo
	CF Code = "CF" // Central African Republic
	CG Code = "CG" // Congo
	CH Code = "CH" // Switzerland
	CI Code = "CI" // Côte d'Ivoire
	CK Code = "CK" // Cook Islands
	CL Code = "CL" // Chile
	CM Code = "CM" // Cameroon
	CN Code = "CN" // China
	CO Code = "CO" // Colombia
	CR Code = "CR" // Costa Rica
	CU Code = "CU" // Cuba
	CV Code = "CV" // Cabo Verde
	CW Code = "CW" // Curaçao
	CX Code = "CX" // Christmas Island
	CY Code = "CY" // Cyprus
	CZ Code = "CZ" // Czechia
	DE Code = "DE" // Germany
	DJ Code = "DJ" // Djibouti
	DK Code = "DK" // Denmark
	DM Code = "DM" // Dominica
	DO Code = "DO" // Dominican Republic
	DZ Code = "DZ" // Algeria
	EC Code = "EC" // Ecuador
	EE Code = "EE" // Estonia
	EG Code = "EG" // Egypt
	EH Code = "EH" // Western Sahara
	ER Code = "ER" // Eritrea
	ES Code = "ES" // Spain
	ET Code = "ET" // Ethiopia
	FI Code = "FI" // Finland
	FJ Code = "FJ" // Fiji
	FK Code = "FK" // Falkland Islands (Malvinas)
	FM Code = "FM" // Micronesia
	FO Code = "FO" // Faroe Islands
	FR Code = "FR" // France
	GA Code = "GA" // Gabon
	GB Code = "GB" // United Kingdom
	GD Code = "GD" // Grenada
	GE Code = "GE" // Georgia
	GF Code = "GF" // French Guiana
	GG Code = "GG" // Guernsey
	GH Code = "GH" // Ghana
	GI Code = "GI" // Gibraltar
	GL Code = "GL" // Greenland
	GM Code = "GM" // Gambia
	GN Code = "GN" // Guinea
	GP Code = "GP" // Guadeloupe
	GQ Code = "GQ" // Equatorial Guinea
	GR Code = "GR" // Greece
	GS Code = "GS" // South Georgia and the South Sandwich Islands
	GT Code = "GT" // Guatemala
	GU Code = "GU" // Guam
	GW Code = "GW" // Guinea-Bissau
	GY Code = "GY" // Guyana
	HK Code = "HK" // Hong Kong
	HM Code = "HM" // Heard Island and McDonald Islands
	HN Code = "HN" // Honduras
	HR Code = "HR" // Croatia
	HT Code = "HT" // Haiti
	HU Code = "HU" // Hungary
	ID Code = "ID" // Indonesia
	IE Code = "IE" // Ireland
	IL Code = "IL" // Israel
	IM Code = "IM" // Isle of Man
	IN Code = "IN" // India
	IO Code = "IO" // British Indian Ocean Territory
	IQ Code = "IQ" // Iraq
	IR Code = "IR" // Iran
	IS Code = "IS" // Iceland
	IT Code = "IT" // Italy
	JE Code = "JE" // Jersey
	JM Code = "JM" // Jamaica
	JO Code = "JO" // Jordan
	JP Code = "JP" // Japan
	KE Code = "KE" // Kenya
	KG Code = "KG" // Kyrgyzstan
	KH Code = "KH" // Cambodia
	KI Code = "KI" // Kiribati
	KM Code = "KM" // Comoros
	KN Code = "KN" // Saint Kitts and Nevis
	KP Code = "KP" // North Korea
	KR Code = "KR" // South Korea
	KW Code = "KW" // Kuwait
	KY Code = "KY" // Cayman Islands
	KZ Code = "KZ" // Kazakhstan
	LA Code = "LA" // Lao People's Democratic Republic
	LB Code = "LB" // Lebanon
	LC Code = "LC" // Saint Lucia
	LI Code = "LI" // Liechtenstein
	LK Code = "LK" // Sri Lanka
	LR Code = "LR" // Liberia
	LS Code = "LS" // Lesotho
	LT Code = "LT" // Lithuania
	LU Code = "LU" // Luxembourg
	LV Code = "LV" // Latvia
	LY Code = "LY" // Libya
	MA Code = "MA" // Morocco
	MC Code = "MC" // Monaco
	MD Code = "MD" // Moldova
	ME Code = "ME" // Montenegro
	MF Code = "MF" // Saint Martin (French part)
	MG Code = "MG" // Madagascar
	MH Code = "MH" // Marshall Islands
	MK Code = "MK" // North Macedonia
	ML Code = "ML" // Mali
	MM Code = "MM" // Myanmar
	MN Code = "MN" // Mongolia
	MO Code = "MO" // Macao
	MP Code = "MP" // Northern Mariana Islands
	MQ Code = "MQ" // Martinique
	MR Code = "MR" // Mauritania
	MS Code = "MS" // Montserrat
	MT Code = "MT" // Malta
	MU Code = "MU" // Mauritius
	MV Code = "MV" // Maldives
	MW Code = "MW" // Malawi
	MX Code = "MX" // Mexico
	MY Code = "MY" // Malaysia
	MZ Code = "MZ" // Mozambique
	NA Code = "NA" // Namibia
	NC Code = "NC" // New Caledonia
	NE Code = "NE" // Niger
	NF Code = "NF" // Norfolk Island
	NG Code = "NG" // Nigeria
	NI Code = "NI" // Nicaragua
	NL Code = "NL" // Netherlands
	NO Code = "NO" // Norway
	NP Code = "NP" // Nepal
	NR Code = "NR" // Nauru
	NU Code = "NU" // Niue
	NZ Code = "NZ" // New Zealand
	OM Code = "OM" // Oman
	PA Code = "PA" // Panama
	PE Code = "PE" // Peru
	PF Code = "PF" // French Polynesia
	PG Code = "PG" // Papua New Guinea
	PH Code = "PH" // Philippines
	PK Code = "PK" // Pakistan
	PL Code = "PL" // Poland
	PM Code = "PM" // Saint Pierre and Miquelon
	PN Code = "PN" // Pitcairn
	PR Code = "PR" // Puerto Rico
	PS Code = "PS" // Palestine
	PT Code = "PT" // Portugal
	PW Code = "PW" // Palau
	PY Code = "PY" // Paraguay
	QA Code = "QA" // Qatar
	RE Code = "RE" // Réunion
	RO Code = "RO" // Romania
	RS Code = "RS" // Serbia
	RU Code = "RU" // Russian Federation
	RW Code = "RW" // Rwanda
	SA Code = "SA" // Saudi Arabia
	SB Code = "SB" // Solomon Islands
	SC Code = "SC" // Seychelles
	SD Code = "SD" // Sudan
	SE Code = "SE" // Sweden
	SG Code = "SG" // Singapore
	SH Code = "SH" // Saint Helena, Ascension and Tristan da Cunha
	SI Code = "SI" // Slovenia
	SJ Code = "SJ" // Svalbard and Jan Mayen
	SK Code = "SK" // Slovakia
	SL Code = "SL" // Sierra Leone
	SM Code = "SM" // San Marino
	SN Code = "SN" // Senegal
	SO Code = "SO" // Somalia
	SR Code = "SR" // Suriname
	SS Code = "SS" // South Sudan
	ST Code = "ST" // Sao Tome and Principe
	SV Code = "SV" // El Salvador
	SX Code = "SX" // Sint Maarten (Dutch part)
	SY Code = "SY" // Syrian Arab Republic
	SZ Code = "SZ" // Eswatini
	TC Code = "TC" // Turks and Caicos Islands
	TD Code = "TD" // Chad
	TF Code = "TF" // French Southern Territories
	TG Code = "TG" // Togo
	TH Code = "TH" // Thailand
	TJ Code = "TJ" // Tajikistan
	TK Code = "TK" // Tokelau
	TL Code = "TL" // Timor-Leste
	TM Code = "TM" // Turkmenistan
	TN Code = "TN" // Tunisia
	TO Code = "TO" // Tonga
	TR Code = "TR" // Türkiye
	TT Code = "TT" // Trinidad and Tobago
	TV Code = "TV" // Tuvalu
	TW Code = "TW" // Taiwan
	TZ Code = "TZ" // Tanzania
	UA Code = "UA" // Ukraine
	UG Code = "UG" // Uganda
	UM Code = "UM" // United States Minor Outlying Islands
	US Code = "US" // United States of America
	UY Code = "UY" // Uruguay
	UZ Code = "UZ" // Uzbekistan
	VA Code = "VA" // Holy See
	VC Code = "VC" // Saint Vincent and the Grenadines
	VE Code = "VE" // Venezuela
	VG Code = "VG" // Virgin Islands (British)
	VI Code = "VI" // Virgin Islands (U.S.)
	VN Code = "VN" // Viet Nam
	VU Code = "VU" // Vanuatu
	WF Code = "WF" // Wallis and Futuna
	WS Code = "WS" // Samoa
	YE Code = "YE" // Yemen
	YT Code = "YT" // Mayotte
	ZA Code = "ZA" // South Africa
	ZM Code = "ZM" // Zambia
	ZW Code = "ZW" // Zimbabwe
)

var names = map[Code]string{
	AD: "Andorra",
	AE: "United Arab Emirates",
	AF: "Afghanistan",
	AG: "Antigua and Barbuda",
	AI: "Anguilla",
	AL: "Albania",
	AM: "Armenia",
	AO: "Angola",
	AQ: "Antarctica",
	AR: "Argentina",
	AS: "American Samoa",
	AT: "Austria",
	AU: "Australia",
	AW: "Aruba",
	AX: "Åland Islands",
	AZ: "Azerbaijan",
	BA: "Bosnia and Herzegovina",
	BB: "Barbados",
	BD: "Bangladesh",
	BE: "Belgium",
	BF: "Burkina Faso",
	BG: "Bulgaria",
	BH: "Bahrain",
	BI: "Burundi",
	BJ: "Benin",
	BL: "Saint Barthélemy",
	BM: "Bermuda",
	BN: "Brunei Darussalam",
	BO: "Bolivia",
	BQ: "Bonaire, Sint Eustatius and Saba",
	BR: "Brazil",
	BS: "Bahamas",
	BT: "Bhutan",
	BV: "Bouvet Island",
	BW: "Botswana",
	BY: "Belarus",
	BZ: "Belize",
	CA: "Canada",
	CC: "Cocos (Keeling) Islands",
	CD: "Democratic Republic of the Congo",
	CF: "Central African Republic",
	CG: "Congo",
	CH: "Switzerland",
	CI: "Côte d'Ivoire",
	CK: "Cook Islands",
	CL: "Chile",
	CM: "Cameroon",
	CN: "China",
	CO: "Colombia",
	CR: "Costa Rica",
	CU: "Cuba",
	CV: "Cabo Verde",
	CW: "Curaçao",
	CX: "Christmas Island",
	CY: "Cyprus",
	CZ: "Czechia",
	DE: "Germany",
	DJ: "Djibouti",
	DK: "Denmark",
	DM: "Dominica",
	DO: "Dominican Republic",
	DZ: "Algeria",
	EC: "Ecuador",
	EE: "Estonia",
	EG: "Egypt",
	EH: "Western Sahara",
	ER: "Eritrea",
	ES: "Spain",
	ET: "Ethiopia",
	FI: "Finland",
	FJ: "Fiji",
	FK: "Falkland Islands (Malvinas)",
	FM: "Micronesia",
	FO: "Faroe Islands",
	FR: "France",
	GA: "Gabon",
	GB: "United Kingdom",
	GD: "Grenada",
	GE: "Georgia",
	GF: "French Guiana",
	GG: "Guernsey",
	GH: "Ghana",
	GI: "Gibraltar",
	GL: "Greenland",
	GM: "Gambia",
	GN: "Guinea",
	GP: "Guadeloupe",
	GQ: "Equatorial Guinea",
	GR: "Greece",
	GS: "South Georgia and the South Sandwich Islands",
	GT: "Guatemala",
	GU: "Guam",
	GW: "Guinea-Bissau",
	GY: "Guyana",
	HK: "Hong Kong",
	HM: "Heard Island and McDonald Islands",
	HN: "Honduras",
	HR: "Croatia",
	HT: "Haiti",
	HU: "Hungary",
	ID: "Indonesia",
	IE: "Ireland",
	IL: "Israel",
	IM: "Isle of Man",
	IN: "India",
	IO: "British Indian Ocean Territory",
	IQ: "Iraq",
	IR: "Iran",
	IS: "Iceland",
	IT: "Italy",
	JE: "Jersey",
	JM: "Jamaica",
	JO: "Jordan",
	JP: "Japan",
	KE: "Kenya",
	KG: "Kyrgyzstan",
	KH: "Cambodia",
	KI: "Kiribati",
	KM: "Comoros",
	KN: "Saint Kitts and Nevis",
	KP: "North Korea",
	KR: "South Korea",
	KW: "Kuwait",
	KY: "Cayman Islands",
	KZ: "Kazakhstan",
	LA: "Lao People's Democratic Republic",
	LB: "Lebanon",
	LC: "Saint Lucia",
	LI: "Liechtenstein",
	LK: "Sri Lanka",
	LR: "Liberia",
	LS: "Lesotho",
	LT: "Lithuania",
	LU: "Luxembourg",
	LV: "Latvia",
	LY: "Libya",
	MA: "Morocco",
	MC: "Monaco",
	MD: "Moldova",
	ME: "Montenegro",
	MF: "Saint Martin (French part)",
	MG: "Madagascar",
	MH: "Marshall Islands",
	MK: "North Macedonia",
	ML: "Mali",
	MM: "Myanmar",
	MN: "Mongolia",
	MO: "Macao",
	MP: "Northern Mariana Islands",
	MQ: "Martinique",
	MR: "Mauritania",
	MS: "Montserrat",
	MT: "Malta",
	MU: "Mauritius",
	MV: "Maldives",
	MW: "Malawi",
	MX: "Mexico",
	MY: "Malaysia",
	MZ: "Mozambique",
	NA: "Namibia",
	NC: "New Caledonia",
	NE: "Niger",
	NF: "Norfolk Island",
	NG: "Nigeria",
	NI: "Nicaragua",
	NL: "Netherlands",
	NO: "Norway",
	NP: "Nepal",
	NR: "Nauru",
	NU: "Niue",
	NZ: "New Zealand",
	OM: "Oman",
	PA: "Panama",
	PE: "Peru",
	PF: "French Polynesia",
	PG: "Papua New Guinea",
	PH: "Philippines",
	PK: "Pakistan",
	PL: "Poland",
	PM: "Saint Pierre and Miquelon",
	PN: "Pitcairn",
	PR: "Puerto Rico",
	PS: "Palestine",
	PT: "Portugal",
	PW: "Palau",
	PY: "Paraguay",
	QA: "Qatar",
	RE: "Réunion",
	RO: "Romania",
	RS: "Serbia",
	RU: "Russian Federation",
	RW: "Rwanda",
	SA: "Saudi Arabia",
	SB: "Solomon Islands",
	SC: "Seychelles",
	SD: "Sudan",
	SE: "Sweden",
	SG: "Singapore",
	SH: "Saint Helena, Ascension and Tristan da Cunha",
	SI: "Slovenia",
	SJ: "Svalbard and Jan Mayen",
	SK: "Slovakia",
	SL: "Sierra Leone",
	SM: "San Marino",
	SN: "Senegal",
	SO: "Somalia",
	SR: "Suriname",
	SS: "South Sudan",
	ST: "Sao Tome and Principe",
	SV: "El Salvador",
	SX: "Sint Maarten (Dutch part)",
	SY: "Syrian Arab Republic",
	SZ: "Eswatini",
	TC: "Turks and Caicos Islands",
	TD: "Chad",
	TF: "French Southern Territories",
	TG: "Togo",
	TH: "Thailand",
	TJ: "Tajikistan",
	TK: "Tokelau",
	TL: "Timor-Leste",
	TM: "Turkmenistan",
	TN: "Tunisia",
	TO: "Tonga",
	TR: "Türkiye",
	TT: "Trinidad and Tobago",
	TV: "Tuvalu",
	TW: "Taiwan",
	TZ: "Tanzania",
	UA: "Ukraine",
	UG: "Uganda",
	UM: "United States Minor Outlying Islands",
	US: "United States of America",
	UY: "Uruguay",
	UZ: "Uzbekistan",
	VA: "Holy See",
	VC: "Saint Vincent and the Grenadines",
	VE: "Venezuela",
	VG: "Virgin Islands (British)",
	VI: "Virgin Islands (U.S.)",
	VN: "Viet Nam",
	VU: "Vanuatu",
	WF: "Wallis and Futuna",
	WS: "Samoa",
	YE: "Yemen",
	YT: "Mayotte",
	ZA: "South Africa",
	ZM: "Zambia",
	ZW: "Zimbabwe",
}
