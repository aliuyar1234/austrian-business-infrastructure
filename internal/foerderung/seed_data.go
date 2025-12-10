package foerderung

// GetSeedDataFromTypeScript returns all 74 Förderungen ported from the TypeScript project
// This data matches the format from D:\Projekte\foerderungsradar\packages\data\foerderungen\*.json
func GetSeedDataFromTypeScript() []SeedFoerderung {
	return []SeedFoerderung{
		// === AWS (12 programs) ===
		{
			ID:             "aws-garantie",
			Name:           "aws Garantie",
			Beschreibung:   "Bundesgarantie für Bankkredite bis 80% des Kreditbetrags (max. Obligo €30 Mio, Laufzeit bis 20 Jahre). Ermöglicht Finanzierung auch ohne ausreichende Sicherheiten.",
			Traeger:        "aws",
			Bundesland:     nil,
			Art:            "garantie",
			MaxBetrag:      30000000,
			Foerderquote:   ptrFloat64(0.8),
			MinProjektkosten: ptrInt(10000),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel", "gross"},
			},
			Themen:          []string{"investition", "gruendung", "innovation"},
			Einreichfrist:   nil,
			Quelle:          "https://www.aws.at/aws-garantie/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Betriebsstätte oder Sitz in Österreich. Positiver Beitrag zur österreichischen Wirtschaft. Antrag vor Projektstart. Für Innovations- und Wachstumsprojekte. First come, first serve."),
			Active:          true,
		},
		{
			ID:             "aws-erp-kredit",
			Name:           "aws ERP-Kredit",
			Beschreibung:   "Zinsgünstiger Kredit von €10.000 bis €30 Mio für Gründung, Modernisierung, Wachstum, Innovation. Bis 100% der Projektkosten finanzierbar, Laufzeit bis 16 Jahre.",
			Traeger:        "aws",
			Bundesland:     nil,
			Art:            "kredit",
			MaxBetrag:      30000000,
			Foerderquote:   ptrFloat64(1.0),
			MinProjektkosten: ptrInt(10000),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel", "gross"},
			},
			Themen:          []string{"investition", "gruendung", "innovation"},
			Einreichfrist:   nil,
			Quelle:          "https://www.aws.at/aws-erp-kredit/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Laufzeit 5,5-16 Jahre mit tilgungsfreier Zeit. Eintrittsgebühr 0,9% (0,5% für Gründer bis 6 Jahre). 99% Bewilligungsquote. Antrag vor Projektstart erforderlich."),
			Active:          true,
		},
		{
			ID:           "aws-eigenkapital",
			Name:         "aws Eigenkapital (Gründungsfonds)",
			Beschreibung: "Beteiligungskapital (Co-Investment) für Start-ups mit skalierbarem Geschäftsmodell. Erstinvest €0,5-1,5 Mio, gesamt bis €5 Mio pro Unternehmen.",
			Traeger:      "aws",
			Bundesland:   nil,
			Art:          "beteiligung",
			MaxBetrag:    5000000,
			MinProjektkosten: ptrInt(500000),
			Zielgruppe: Zielgruppe{
				Groessen:      []string{"epu", "kleinst", "klein"},
				MaxAlterJahre: ptrInt(7),
			},
			Themen:          []string{"gruendung", "innovation", "export"},
			Einreichfrist:   nil,
			Quelle:          "https://www.aws.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Zielgruppe: Start-ups unter 7 Jahren mit skalierbarem Geschäftsmodell. Fokus auf Markteinführung, Vertriebsaufbau, Internationalisierung. Co-Investment mit privaten Investoren."),
			Active:          true,
		},
		{
			ID:           "aws-innovationsschutz",
			Name:         "aws Innovationsschutz (Advanced IP)",
			Beschreibung: "Coaching und Zuschuss für Schutz geistigen Eigentums (Patente, Know-how). Individuelle IP-Beratung und finanzielle Förderung zur Umsetzung.",
			Traeger:      "aws",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    50000,
			Foerderquote: ptrFloat64(0.5),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"innovation"},
			Einreichfrist:   nil,
			Quelle:          "https://www.aws.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Schwerpunkt GreenTech. Für Schutzrechte, FTO-Analysen (Freedom-to-operate). IP-Strategie-Coaching inklusive. Start-ups und KMU."),
			Active:          true,
		},
		{
			ID:           "aws-wachstumsinvestition",
			Name:         "aws Wachstumsinvestition",
			Beschreibung: "Zuschuss für Erweiterungsprojekte, i.d.R. €300-400k, max. ~€1 Mio (bis 50% der Kosten). Für Maschinen, Anlagen, Prototypen.",
			Traeger:      "aws",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    1000000,
			Foerderquote: ptrFloat64(0.5),
			MinProjektkosten: ptrInt(100000),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"investition", "innovation"},
			Einreichfrist:   nil,
			Quelle:          "https://www.aws.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Nur in Kombination mit aws ERP-Kredit. Für Wachstums- und Innovationsprojekte. First come, first serve. Großunternehmen ausnahmsweise möglich."),
			Active:          true,
		},
		{
			ID:           "aws-digitalisierung",
			Name:         "aws Digitalisierung",
			Beschreibung: "Zuschuss für Digitalisierungsprojekte bis €150.000 (max. 30-50% der Kosten). Für digitale Produkte, Prozesse, E-Commerce, KI-Initiativen.",
			Traeger:      "aws",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    150000,
			Foerderquote: ptrFloat64(0.5),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"digitalisierung", "innovation"},
			Einreichfrist:   nil,
			Quelle:          "https://www.aws.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Primär KMU, teils Großunternehmen. Digitalisierung von Produkten, Prozessen, E-Commerce. Inkl. KI-Initiativen. Teils über thematische Calls."),
			Active:          true,
		},
		{
			ID:           "aws-energie-klima",
			Name:         "aws Energie & Klima",
			Beschreibung: "Zuschuss bis €400.000 (bis 45% Förderquote) für Umwelt- und Energieprojekte. Einführung Energiemanagement, Industrialisierung Umwelttechnologien.",
			Traeger:      "aws",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    400000,
			Foerderquote: ptrFloat64(0.45),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"nachhaltigkeit", "innovation"},
			Einreichfrist:   ptr("2026-12-31"),
			Quelle:          "https://www.aws.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Zielgruppe: Start-ups und KMU. Laufend bis 31.12.2026. Fertigungsüberleitung von Umwelttechnologien."),
			Active:          true,
		},
		{
			ID:           "aws-landwirtschaft",
			Name:         "aws Verarbeitung landwirtschaftlicher Erzeugnisse",
			Beschreibung: "Zuschuss bis €1 Mio (10-30% der Kosten) für Anlagen zur Verarbeitung/Vermarktung landwirtschaftlicher Produkte.",
			Traeger:      "aws",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    1000000,
			Foerderquote: ptrFloat64(0.3),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
				Branchen: []string{"Landwirtschaft", "Lebensmittelverarbeitung"},
			},
			Themen:          []string{"investition", "nachhaltigkeit"},
			Einreichfrist:   nil,
			Quelle:          "https://www.aws.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("KMU, landwirtschaftliche Betriebe/Vereine in Agrar-Verarbeitung. Qualität, Nachhaltigkeit, Effizienz steigern. Über aws Fördermanager."),
			Active:          true,
		},
		{
			ID:           "aws-first-incubator",
			Name:         "aws First Incubator",
			Beschreibung: "Stipendium für angehende Gründer*innen: 12 Monate Coaching/Mentoring, Workshops, Co-Working, Projektbudget bis €55.000 (90% Zuschuss bis €49.000).",
			Traeger:      "aws",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    49000,
			Foerderquote: ptrFloat64(0.9),
			Zielgruppe: Zielgruppe{
				Groessen:      []string{"epu"},
				MaxAlterJahre: ptrInt(1),
			},
			Themen:          []string{"gruendung", "innovation"},
			Einreichfrist:   nil,
			Quelle:          "https://www.aws.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Ideenphase, <4 Teammitglieder. Von Geschäftsidee bis MVP/Markteintritt. 2 Calls pro Jahr (2026: Call #1 für 18-30 Jahre, Call #2 offen)."),
			Active:          true,
		},
		{
			ID:           "aws-preseed",
			Name:         "aws Preseed",
			Beschreibung: "Zuschuss für Vorgründungsphase/Proof of Concept. Innovative Solutions bis €89k (€100k mit Gründerinnen-Bonus), Deep Tech bis €267k (€300k mit Bonus).",
			Traeger:      "aws",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    300000,
			Foerderquote: ptrFloat64(0.8),
			Zielgruppe: Zielgruppe{
				Groessen:      []string{"epu", "kleinst"},
				MaxAlterJahre: ptrInt(1),
			},
			Themen:          []string{"gruendung", "innovation"},
			Einreichfrist:   nil,
			Quelle:          "https://www.aws.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Für F&E-basierte Start-ups. Innovative Solutions: max €100k. Deep Tech: max €300k. Laufende Einreichung, Jury-Entscheide mehrmals jährlich."),
			Active:          true,
		},
		{
			ID:           "aws-seedfinancing",
			Name:         "aws Seedfinancing",
			Beschreibung: "Zuschuss bis 5 Jahre nach Gründung für Marktreife. Innovative Solutions bis €356k (€400k mit Bonus), Deep Tech bis €889k (€1 Mio mit Bonus).",
			Traeger:      "aws",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    1000000,
			Foerderquote: ptrFloat64(0.8),
			Zielgruppe: Zielgruppe{
				Groessen:      []string{"epu", "kleinst", "klein"},
				MaxAlterJahre: ptrInt(5),
			},
			Themen:          []string{"gruendung", "innovation"},
			Einreichfrist:   nil,
			Quelle:          "https://www.aws.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Für Deep-Tech und innovative Start-ups. Meist 80% der Kosten. Jury-Verfahren. Gründerinnen-Bonus verfügbar."),
			Active:          true,
		},

		// === FFG (9 programs) ===
		{
			ID:           "ffg-basisprogramm",
			Name:         "FFG Basisprogramm",
			Beschreibung: "Zuschuss oder zinsgünstiges Darlehen für F&E-Projekte ohne Themenvorgabe (bottom-up). Für alle Unternehmen, typisch bis €3 Mio.",
			Traeger:      "ffg",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    3000000,
			Foerderquote: ptrFloat64(0.6),
			MinProjektkosten: ptrInt(100000),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel", "gross"},
			},
			Themen:          []string{"innovation"},
			Einreichfrist:   nil,
			Quelle:          "https://www.ffg.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Marktnahe Forschung & Entwicklung. Keine Themenvorgabe. Laufende Einreichung mit Juryverfahren. BMWET/BMIMI. Industrielle Forschung bis 60% für KMU."),
			Active:          true,
		},
		{
			ID:           "ffg-basisprogramm-kleinprojekt",
			Name:         "FFG Basisprogramm Kleinprojekt",
			Beschreibung: "Zuschuss bis €90.000 für kleinere F&E-Projekte. Vereinfachter Antragsprozess für KMU und Start-ups.",
			Traeger:      "ffg",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    90000,
			Foerderquote: ptrFloat64(0.6),
			MinProjektkosten: ptrInt(25000),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"innovation"},
			Einreichfrist:   nil,
			Quelle:          "https://www.ffg.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Für F&E-Vorhaben geringeren Umfangs. KMU und Start-ups bevorzugt. Schnellere Bearbeitung. Laufend 2025."),
			Active:          true,
		},
		{
			ID:           "ffg-markteinstieg",
			Name:         "FFG Markt.Einstieg",
			Beschreibung: "Zuschuss bis €120.000 (De-minimis) für erstmaligen Markteintritt. Unterstützung bei Umsetzung und Vermarktung einer Entwicklung.",
			Traeger:      "ffg",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    120000,
			Foerderquote: ptrFloat64(0.5),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"innovation", "export"},
			Einreichfrist:   ptr("2025-12-31"),
			Quelle:          "https://www.ffg.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Neu seit 09/2025 (Nachfolgeprogramm von Markt.Start). Erstmaliger Markteintritt. Call bis 31.12.2025. BMWET."),
			Active:          true,
		},
		{
			ID:           "ffg-collective-research",
			Name:         "FFG Collective Research",
			Beschreibung: "Zuschuss bis €325.000 für vorwettbewerbliche Branchen-F&E-Projekte im Verbund (mind. 3 Unternehmen + Forschung).",
			Traeger:      "ffg",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    325000,
			Foerderquote: ptrFloat64(0.7),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"kleinst", "klein", "mittel", "gross"},
			},
			Themen:          []string{"innovation"},
			Einreichfrist:   nil,
			Quelle:          "https://www.ffg.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Gemeinschaftliche Forschung über Cluster. Vorwettbewerbliche Projekte. Mindestens 3 Unternehmen als Konsortium. Laufend 2025."),
			Active:          true,
		},
		{
			ID:           "ffg-spin-off-fellowship",
			Name:         "FFG Spin-off Fellowship",
			Beschreibung: "Stipendium bis €500.000 für Ausgründungen aus Forschungsinstituten. Für Forschende (Uni/FH) und Pre-Start-ups.",
			Traeger:      "ffg",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    500000,
			Zielgruppe: Zielgruppe{
				Groessen:      []string{"epu", "kleinst"},
				MaxAlterJahre: ptrInt(2),
			},
			Themen:          []string{"gruendung", "innovation"},
			Einreichfrist:   ptr("2026-01-21"),
			Quelle:          "https://www.ffg.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("BMBWF. Personalkosten + Sachmittel. Ausgründung aus Forschungsinstituten. Einreichfrist 21.01.2026."),
			Active:          true,
		},
		{
			ID:           "ffg-diversity-cheque",
			Name:         "FFG Diversity Cheque",
			Beschreibung: "Pauschalzuschuss €10.000 für Diversitätsmaßnahmen im Betrieb (Gleichstellung, Inklusion).",
			Traeger:      "ffg",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    10000,
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"personal"},
			Einreichfrist:   ptr("2026-05-29"),
			Quelle:          "https://www.ffg.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("KMU. Fixbetrag €10.000. Diversität, Gleichstellung, Inklusion. Einreichung bis 29.05.2026."),
			Active:          true,
		},
		{
			ID:           "ffg-diversitec",
			Name:         "FFG DIVERSITEC",
			Beschreibung: "Zuschuss bis €50.000 für Innovation durch Diversität. Organisationsentwicklung für Diversität & Inklusion.",
			Traeger:      "ffg",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    50000,
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel", "gross"},
			},
			Themen:          []string{"personal", "innovation"},
			Einreichfrist:   ptr("2026-06-30"),
			Quelle:          "https://www.ffg.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Organisationsentwicklung für Diversität. 01.07.2025 - 30.06.2026."),
			Active:          true,
		},
		{
			ID:           "ffg-industrienahe-dissertationen",
			Name:         "FFG Industrienahe Dissertationen",
			Beschreibung: "Zuschuss €110.000 (Pauschale) für PhD-Projekte in Unternehmen. Anwendungsorientierte Forschungsarbeiten.",
			Traeger:      "ffg",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    110000,
			Zielgruppe: Zielgruppe{
				Groessen: []string{"klein", "mittel", "gross"},
			},
			Themen:          []string{"innovation", "personal"},
			Einreichfrist:   ptr("2026-09-30"),
			Quelle:          "https://www.ffg.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("FFG/Land OÖ. Unternehmen mit F&E + außeruniversitäre Forschung. Call bis 30.09.2026."),
			Active:          true,
		},

		// === WKO (3 programs) ===
		{
			ID:           "wko-kmu-digital",
			Name:         "KMU.DIGITAL",
			Beschreibung: "Beratungs- und Umsetzungsförderung für Digitalisierung. Beratung bis 80% (max. €1.400), Umsetzung 30% (max. €6.000).",
			Traeger:      "wko",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    7400,
			Foerderquote: ptrFloat64(0.3),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"digitalisierung"},
			Einreichfrist:   nil,
			Quelle:          "https://www.wko.at/foerderungen",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Status-/Potenzialanalyse 80% (max. €400). Strategieberatung 50% (max. €1.000). Umsetzung 30% (max. €6.000 pro Unternehmen). Programmperiode bis 2026."),
			Active:          true,
		},
		{
			ID:           "wko-kmu-digital-green",
			Name:         "KMU.DIGITAL GREEN",
			Beschreibung: "Nachhaltigkeitsmodul von KMU.DIGITAL. Fokus auf Energieeffizienz, Kreislaufwirtschaft, nachhaltige Digitalisierung.",
			Traeger:      "wko",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    7400,
			Foerderquote: ptrFloat64(0.3),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"digitalisierung", "nachhaltigkeit"},
			Einreichfrist:   nil,
			Quelle:          "https://www.wko.at/foerderungen",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("GREEN-Modul mit Fokus auf Energieeffizienz, Kreislaufwirtschaft. Kombination mit KMU.DIGITAL möglich. Programmperiode bis 2026."),
			Active:          true,
		},
		{
			ID:           "wko-go-international",
			Name:         "go-international (Internationalisierungsschecks)",
			Beschreibung: "Schecks für Exportaktivitäten: bis 50% der Markteintrittskosten (typ. bis €10.000). Diverse Module verfügbar.",
			Traeger:      "wko",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    10000,
			Foerderquote: ptrFloat64(0.5),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"export"},
			Einreichfrist:   ptr("2027-03-31"),
			Quelle:          "https://www.wko.at/foerderungen",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Module: Internationalisierungsscheck, Digital Marketing Scheck, Bildungsscheck, Sourcing-Scheck, Projektgeschäft-Scheck. Programm April 2023 - März 2027 mit €51,2 Mio Budget."),
			Active:          true,
		},

		// === AMS (4 programs) ===
		{
			ID:           "ams-eingliederungsbeihilfe",
			Name:         "AMS Eingliederungsbeihilfe (Come-back)",
			Beschreibung: "Lohnkostenzuschuss bei Einstellung arbeitsloser Personen. 30-66% des Brutto-Lohns für 6-12 Monate. Für Langzeitarbeitslose und 50+.",
			Traeger:      "ams",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    25000,
			Foerderquote: ptrFloat64(0.66),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel", "gross"},
			},
			Themen:          []string{"personal"},
			Einreichfrist:   nil,
			Quelle:          "https://www.ams.at/unternehmen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Langzeitarbeitslose oder ältere Arbeitsuchende (50+). Jüngere ab 6 Monate AMS-Vormerkung. Prozentsatz und Dauer individuell vereinbart. Beratungsgespräch mit AMS vor Dienstbeginn erforderlich."),
			Active:          true,
		},
		{
			ID:           "ams-epu-erste-arbeitskraft",
			Name:         "AMS EPU: Förderung erste Arbeitskraft",
			Beschreibung: "25% Erstattung des Bruttogehalts der ersten Vollzeitkraft für max. 12 Monate. Für Ein-Personen-Unternehmen.",
			Traeger:      "ams",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    15000,
			Foerderquote: ptrFloat64(0.25),
			Zielgruppe: Zielgruppe{
				Groessen:      []string{"epu"},
				MinAlterJahre: ptrInt(0),
			},
			Themen:          []string{"personal", "gruendung"},
			Einreichfrist:   nil,
			Quelle:          "https://www.ams.at/unternehmen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("EPU seit >3 Monaten in GSVG versichert. Mitarbeiter*in mind. 50% Beschäftigung, zuvor 2 Wochen arbeitslos. Antrag binnen 6 Wochen nach Einstellung. Gedeckelt mit ASVG-Höchstbeitragsgrundlage."),
			Active:          true,
		},
		{
			ID:           "ams-lehrlingsausbildung",
			Name:         "AMS Lehrlingsausbildungs-Förderung",
			Beschreibung: "Pauschalzuschüsse pro Lehrmonat für förderbedürftige Gruppen: Mädchen in Männerberufen, benachteiligte Jugendliche, Ü18-Lehrlinge.",
			Traeger:      "ams",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    20000,
			Foerderquote: ptrFloat64(0.4),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel", "gross"},
			},
			Themen:          []string{"personal"},
			Einreichfrist:   nil,
			Quelle:          "https://www.ams.at/unternehmen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Zielgruppen: Mädchen in Männerberufen, benachteiligte Jugendliche, verlängerte Lehre/Teilqualifikation, erwachsene Lehrlinge (Ü18). Beratung mit AMS vor Lehrstart."),
			Active:          true,
		},
		{
			ID:           "ams-qualifizierung",
			Name:         "AMS Qualifizierungsförderung für Beschäftigte",
			Beschreibung: "Bis zu 50% Zuschuss für Weiterbildungskosten älterer oder niedrigqualifizierter Beschäftigter.",
			Traeger:      "ams",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    10000,
			Foerderquote: ptrFloat64(0.5),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"personal"},
			Einreichfrist:   nil,
			Quelle:          "https://www.ams.at/unternehmen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Für Weiterbildungen über 16 Stunden. Höhere Förderung für Geringqualifizierte, Ältere, Wiedereinsteigende. Arbeitsmarktrelevante Qualifikationen."),
			Active:          true,
		},

		// === OeKB (3 programs) ===
		{
			ID:           "oekb-exportfonds-kredit",
			Name:         "OeKB Exportfonds-Kredit",
			Beschreibung: "Revolvierender Betriebsmittelkredit bis 30% des Jahresexportumsatzes. Sehr niedrige Zinsen durch staatliche Refinanzierung.",
			Traeger:      "oekb",
			Bundesland:   nil,
			Art:          "kredit",
			MaxBetrag:    10000000,
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"export"},
			Einreichfrist:   nil,
			Quelle:          "https://www.oekb.at/exportservices/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("KMU mit Exportumsatz. Bis 30% des Jahresexportumsatzes (14% bei Tourismus). Vorfinanzierung von Exportaufträgen, Working Capital. Bund übernimmt Ausfallsrisiko teilweise."),
			Active:          true,
		},
		{
			ID:           "oekb-exporthaftungen",
			Name:         "OeKB Exporthaftungen & Finanzierungen",
			Beschreibung: "Bundesgarantien (85-95% Deckung) + zinsgünstige Exportkredite für Einzelexportgeschäfte mit langfristigem Zahlungsziel.",
			Traeger:      "oekb",
			Bundesland:   nil,
			Art:          "garantie",
			MaxBetrag:    50000000,
			Foerderquote: ptrFloat64(0.95),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"klein", "mittel", "gross"},
			},
			Themen:          []string{"export"},
			Einreichfrist:   nil,
			Quelle:          "https://www.oekb.at/exportservices/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Risikoabdeckung schwieriger Märkte. Für Investitionsgüterexport, Großprojekte. OECD-Konditionen. Laufend über Hausbank."),
			Active:          true,
		},
		{
			ID:           "oekb-exportinvest",
			Name:         "OeKB Exportinvest",
			Beschreibung: "Zinsgünstiger Kredit für heimische Investitionen zur Exportsteigerung. Ab €2 Mio Investitionsvolumen.",
			Traeger:      "oekb",
			Bundesland:   nil,
			Art:          "kredit",
			MaxBetrag:    20000000,
			Zielgruppe: Zielgruppe{
				Groessen: []string{"klein", "mittel", "gross"},
			},
			Themen:          []string{"export", "investition"},
			Einreichfrist:   nil,
			Quelle:          "https://www.oekb.at/exportservices/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Exportquote >= 20% erforderlich. +20% zusätzlicher Kredit für grüne Investitionen (Exportinvest Green). Auch für Auslandsbeteiligungen/Niederlassungen."),
			Active:          true,
		},

		// === EU (3 programs) ===
		{
			ID:           "eu-horizon-europe",
			Name:         "Horizon Europe",
			Beschreibung: "EU-Rahmenprogramm für Forschung & Innovation. Grants bis 70-100% für Verbundprojekte, Fellowships etc. Budget ~€95 Mrd EU-weit.",
			Traeger:      "eu",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    5000000,
			Foerderquote: ptrFloat64(1.0),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel", "gross"},
			},
			Themen:          []string{"innovation", "nachhaltigkeit", "digitalisierung"},
			Einreichfrist:   nil,
			Quelle:          "https://www.ffg.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("EU-Rahmenprogramm 2021-27. Oft als Teil eines Konsortiums. Schwerpunkte: Klimaschutz, Digitalisierung, Gesundheit, Industrie 4.0. FFG als nationale Kontaktstelle (NCP). Mission Calls, EIC Accelerator für Start-ups."),
			Active:          true,
		},
		{
			ID:           "eu-efre",
			Name:         "EFRE – Europäischer Fonds für regionale Entwicklung",
			Beschreibung: "EU-Strukturfonds für regionale Wettbewerbsfähigkeit, Innovation, CO2-Reduktion. Kofinanziert viele Landesprogramme.",
			Traeger:      "eu",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    1000000,
			Foerderquote: ptrFloat64(0.5),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"innovation", "nachhaltigkeit", "digitalisierung"},
			Einreichfrist:   ptr("2027-12-31"),
			Quelle:          "https://www.ffg.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("IWB-Programm (Investitionen in Wachstum & Beschäftigung). Kofinanziert Landesprogramme (ecoplus NÖ, WAW Wien, SFG Steiermark). Laufend bis 2027."),
			Active:          true,
		},
		{
			ID:           "eu-esf-plus",
			Name:         "ESF+ – Europäischer Sozialfonds Plus",
			Beschreibung: "EU-Fonds für Beschäftigung & Soziales. ~€409 Mio für Österreich 2021-27. Kofinanziert Qualifizierungsprojekte.",
			Traeger:      "eu",
			Bundesland:   nil,
			Art:          "zuschuss",
			MaxBetrag:    500000,
			Foerderquote: ptrFloat64(0.7),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel", "gross"},
			},
			Themen:          []string{"personal"},
			Einreichfrist:   ptr("2027-12-31"),
			Quelle:          "https://www.esf.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Weiterbildungsverbünde, Frauen in Handwerk/Technik. Bis 50-70% Kostenzuschuss für betriebliche Weiterbildung. AMS-Programme wie Qualifizierungsförderung. Projektabhängig über Sozialministeriumservice."),
			Active:          true,
		},

		// === Wien (8 programs) ===
		{
			ID:           "wien-innovation",
			Name:         "Wirtschaftsagentur Wien – Innovationsförderung",
			Beschreibung: "Zuschuss bis €500.000 (30-45% der Kosten) für F&E- und Innovationsprojekte. Jury-Auswahl mit Bonuspunkten für Klima, Beschäftigung, Diversität.",
			Traeger:      "land",
			Bundesland:   ptr("wien"),
			Art:          "zuschuss",
			MaxBetrag:    500000,
			Foerderquote: ptrFloat64(0.45),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"innovation"},
			Einreichfrist:   nil,
			Quelle:          "https://wirtschaftsagentur.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Neue Produkte, Verfahren, Dienstleistungen in Wien. Bonuspunkte für Klimaschutz, Beschäftigung, Diversität. Stichtage mehrmals jährlich."),
			Active:          true,
		},
		{
			ID:           "wien-creative-project",
			Name:         "Wirtschaftsagentur Wien – Creative Project",
			Beschreibung: "Zuschuss bis €150.000 (~50% Quote) für kreative Projekte. Design, Medien, Games, Mode.",
			Traeger:      "land",
			Bundesland:   ptr("wien"),
			Art:          "zuschuss",
			MaxBetrag:    150000,
			Foerderquote: ptrFloat64(0.5),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
				Branchen: []string{"Kreativwirtschaft", "Design", "Medien", "Games", "Mode"},
			},
			Themen:          []string{"innovation"},
			Einreichfrist:   nil,
			Quelle:          "https://wirtschaftsagentur.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Unternehmen der Kreativwirtschaft. Für kreative Projekte und Produktentwicklung. Calls jährlich."),
			Active:          true,
		},
		{
			ID:           "wien-creative-markteintritt",
			Name:         "Wirtschaftsagentur Wien – Creative Markteintritt",
			Beschreibung: "Zuschuss bis €50.000 (~50% Quote) für erste internationale Schritte von Kreativwirtschaftsunternehmen.",
			Traeger:      "land",
			Bundesland:   ptr("wien"),
			Art:          "zuschuss",
			MaxBetrag:    50000,
			Foerderquote: ptrFloat64(0.5),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
				Branchen: []string{"Kreativwirtschaft", "Design", "Medien", "Games", "Mode"},
			},
			Themen:          []string{"export", "innovation"},
			Einreichfrist:   nil,
			Quelle:          "https://wirtschaftsagentur.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Kreativwirtschaft-Unternehmen. Für internationale Expansion. Calls jährlich."),
			Active:          true,
		},
		{
			ID:           "wien-digitalisierung",
			Name:         "Wirtschaftsagentur Wien – Digitalisierung",
			Beschreibung: "Zuschuss bis 50% (max. €50.000) für Digitalisierungsprojekte. End-to-End Digitalisierung, nicht nur Hardware/Webshop.",
			Traeger:      "land",
			Bundesland:   ptr("wien"),
			Art:          "zuschuss",
			MaxBetrag:    50000,
			Foerderquote: ptrFloat64(0.5),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"digitalisierung"},
			Einreichfrist:   nil,
			Quelle:          "https://wirtschaftsagentur.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Wiener KMU mit Betriebsstätte >1 Jahr. Digitale Transformation von Prozessen. Programm 2024-2026, Wettbewerbsprinzip."),
			Active:          true,
		},
		{
			ID:           "wien-internationaler-markteintritt",
			Name:         "Wirtschaftsagentur Wien – Internationaler Markteintritt",
			Beschreibung: "Zuschuss ~50% (bis ~€20.000) für Auslandsexpansion. Messebeteiligungen, Marketing, Exportberatung.",
			Traeger:      "land",
			Bundesland:   ptr("wien"),
			Art:          "zuschuss",
			MaxBetrag:    20000,
			Foerderquote: ptrFloat64(0.5),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"export"},
			Einreichfrist:   nil,
			Quelle:          "https://wirtschaftsagentur.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Wiener Unternehmen. Messebeteiligungen, Marketing im Ausland, Exportberatung. Call-basiert."),
			Active:          true,
		},
		{
			ID:           "wien-lebensqualitaet",
			Name:         "Wirtschaftsagentur Wien – Lebensqualität",
			Beschreibung: "Zuschuss bis ~€100.000 für sozial-innovative Projekte zur Verbesserung urbaner Lebensqualität.",
			Traeger:      "land",
			Bundesland:   ptr("wien"),
			Art:          "zuschuss",
			MaxBetrag:    100000,
			Foerderquote: ptrFloat64(0.5),
			Zielgruppe: Zielgruppe{
				Groessen:      []string{"epu", "kleinst", "klein"},
				MaxAlterJahre: ptrInt(5),
			},
			Themen:          []string{"innovation", "nachhaltigkeit"},
			Einreichfrist:   nil,
			Quelle:          "https://wirtschaftsagentur.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Kleine Unternehmen & Start-ups. Soziale Innovationen für urbane Lebensqualität."),
			Active:          true,
		},
		{
			ID:           "wien-graetzl-initiative",
			Name:         "Wirtschaftsagentur Wien – Grätzl-Initiative",
			Beschreibung: "Zuschüsse für Nahversorger und Geschäftsansiedlungen in Stadtteilen.",
			Traeger:      "land",
			Bundesland:   ptr("wien"),
			Art:          "zuschuss",
			MaxBetrag:    50000,
			Foerderquote: ptrFloat64(0.4),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein"},
			},
			Themen:          []string{"investition", "gruendung"},
			Einreichfrist:   nil,
			Quelle:          "https://wirtschaftsagentur.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Geschäftsbelebung in Stadtteilen. Nahversorger bevorzugt."),
			Active:          true,
		},
		{
			ID:           "wien-planet-fund",
			Name:         "Vienna Planet Fund",
			Beschreibung: "Förderung für klimarelevante Innovationsprojekte in Wien. CO2-Reduktion, Ressourceneffizienz.",
			Traeger:      "land",
			Bundesland:   ptr("wien"),
			Art:          "zuschuss",
			MaxBetrag:    250000,
			Foerderquote: ptrFloat64(0.5),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"nachhaltigkeit", "innovation"},
			Einreichfrist:   nil,
			Quelle:          "https://wirtschaftsagentur.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Klimaschutz-Projekte. Laufend 2024-25. CO2-Reduktion oder Ressourceneffizienz nachweisbar."),
			Active:          true,
		},

		// === Niederösterreich (3 programs) ===
		{
			ID:           "noe-regionalfoerderung",
			Name:         "ecoplus Regionalförderung",
			Beschreibung: "Zuschuss oder zinsloses Darlehen für wirtschaftsnahe Infrastruktur. Kleine Unternehmen bis 20%, mittlere bis 10%.",
			Traeger:      "land",
			Bundesland:   ptr("niederoesterreich"),
			Art:          "zuschuss",
			MaxBetrag:    200000,
			Foerderquote: ptrFloat64(0.2),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"investition"},
			Einreichfrist:   nil,
			Quelle:          "https://www.ecoplus.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("EU (IWB/EFRE) kofinanziert. Touristische und wirtschaftliche Infrastrukturprojekte (Ausflugsziele, Radwege). Großunternehmen nicht gefördert."),
			Active:          true,
		},
		{
			ID:           "noe-technologie-innovation",
			Name:         "NÖ Technologie- und Innovationsförderung",
			Beschreibung: "Zuschüsse für F&E-Projekte in Kooperation mit FFG. Land übernimmt Landesanteil.",
			Traeger:      "land",
			Bundesland:   ptr("niederoesterreich"),
			Art:          "zuschuss",
			MaxBetrag:    500000,
			Foerderquote: ptrFloat64(0.5),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"innovation"},
			Einreichfrist:   nil,
			Quelle:          "https://www.ecoplus.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Kofinanzierung zu FFG-Projekten. EFRE-Kofinanzierung über ecoplus."),
			Active:          true,
		},
		{
			ID:           "noe-wachstumsprogramm",
			Name:         "NÖ Wachstumsprogramm",
			Beschreibung: "Zinsgünstige ERP-Kredite und Zuschüsse für KMU-Investitionen in Niederösterreich.",
			Traeger:      "land",
			Bundesland:   ptr("niederoesterreich"),
			Art:          "zuschuss",
			MaxBetrag:    300000,
			Foerderquote: ptrFloat64(0.15),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"investition", "gruendung"},
			Einreichfrist:   nil,
			Quelle:          "https://www.ecoplus.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Bundesprogramme mit Kofinanzierung durch Landesmittel. Beratung über ecoplus Regionalmanagement."),
			Active:          true,
		},

		// === Steiermark (9 programs) ===
		{
			ID:           "stmk-ideen-reich-xl",
			Name:         "SFG Ideen!Reich XL",
			Beschreibung: "Innovationszuschuss bis €75.000 für größere Innovationsprojekte. Entwicklung neuer Produkte/Dienstleistungen.",
			Traeger:      "land",
			Bundesland:   ptr("steiermark"),
			Art:          "zuschuss",
			MaxBetrag:    75000,
			Foerderquote: ptrFloat64(0.4),
			MinProjektkosten: ptrInt(14000),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"innovation"},
			Einreichfrist:   nil,
			Quelle:          "https://www.sfg.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Modul XL für größere Innovationsprojekte. Vom Konzept bis Prototyp oder Marktreife. Mehrere Cut-offs pro Jahr."),
			Active:          true,
		},
		{
			ID:           "stmk-ideen-reich-xs",
			Name:         "SFG Ideen!Reich XS",
			Beschreibung: "Innovationszuschuss bis €14.000 für kleinere Innovationsprojekte.",
			Traeger:      "land",
			Bundesland:   ptr("steiermark"),
			Art:          "zuschuss",
			MaxBetrag:    14000,
			Foerderquote: ptrFloat64(0.4),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"innovation"},
			Einreichfrist:   nil,
			Quelle:          "https://www.sfg.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Modul XS für kleinere Innovationsprojekte. Schnellere Bearbeitung."),
			Active:          true,
		},
		{
			ID:           "stmk-green-invest",
			Name:         "SFG Green!Invest",
			Beschreibung: "Investitionszuschuss bis 35% für 'Green Deal'-konforme Investitionen. CO2-Reduktion, Kreislaufwirtschaft.",
			Traeger:      "land",
			Bundesland:   ptr("steiermark"),
			Art:          "zuschuss",
			MaxBetrag:    300000,
			Foerderquote: ptrFloat64(0.35),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"nachhaltigkeit", "investition"},
			Einreichfrist:   nil,
			Quelle:          "https://www.sfg.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Produzierende Unternehmen/Dienstleister. Langfristig tragfähige grüne Geschäftsfelder. Call-Verfahren."),
			Active:          true,
		},
		{
			ID:           "stmk-start-klar",
			Name:         "SFG Start!Klar",
			Beschreibung: "Kombinierte Förderung (Investition + Beratung) bis €37.500 (~50%) für Start-ups bis 5 Jahre.",
			Traeger:      "land",
			Bundesland:   ptr("steiermark"),
			Art:          "zuschuss",
			MaxBetrag:    37500,
			Foerderquote: ptrFloat64(0.5),
			Zielgruppe: Zielgruppe{
				Groessen:      []string{"epu", "kleinst", "klein"},
				MaxAlterJahre: ptrInt(5),
			},
			Themen:          []string{"gruendung", "investition"},
			Einreichfrist:   nil,
			Quelle:          "https://www.sfg.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Anschaffungen (Maschinen, IT) und Beratungen (BWL, Recht, Technologie, Markt). Bis 50% der Kosten."),
			Active:          true,
		},
		{
			ID:           "stmk-start-klar-plus",
			Name:         "SFG Start!Klar plus",
			Beschreibung: "80% Zuschuss bis €100.000 für investor-ready Start-ups. Vorbereitung auf erste große Finanzierungsrunde.",
			Traeger:      "land",
			Bundesland:   ptr("steiermark"),
			Art:          "zuschuss",
			MaxBetrag:    100000,
			Foerderquote: ptrFloat64(0.8),
			Zielgruppe: Zielgruppe{
				Groessen:      []string{"epu", "kleinst", "klein"},
				MaxAlterJahre: ptrInt(5),
			},
			Themen:          []string{"gruendung", "innovation"},
			Einreichfrist:   nil,
			Quelle:          "https://www.sfg.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Für Start-ups in der Wachstumsphase. Vorbereitung auf Skalierung und Investorensuche."),
			Active:          true,
		},
		{
			ID:           "stmk-steiermark-bonus",
			Name:         "SFG Steiermark!Bonus",
			Beschreibung: "FFG-Basisprogramm Top-up: +20% Darlehenserhöhung für steirische Unternehmen mit bewilligtem FFG-Projekt.",
			Traeger:      "land",
			Bundesland:   ptr("steiermark"),
			Art:          "kredit",
			MaxBetrag:    600000,
			Foerderquote: ptrFloat64(0.2),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel", "gross"},
			},
			Themen:          []string{"innovation"},
			Einreichfrist:   nil,
			Quelle:          "https://www.sfg.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Voraussetzung: bewilligtes FFG-Basisprogramm-Darlehen. SFG stockt um 20% auf. Mehr Liquidität für F&E-Projekte."),
			Active:          true,
		},
		{
			ID:           "stmk-cyber-sicher",
			Name:         "SFG Cyber!Sicher",
			Beschreibung: "Zuschuss bis 30% (max. €15.000) für IT- und Cybersicherheit. Audit, Zertifizierung, Schulung, Software.",
			Traeger:      "land",
			Bundesland:   ptr("steiermark"),
			Art:          "zuschuss",
			MaxBetrag:    15000,
			Foerderquote: ptrFloat64(0.3),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"digitalisierung"},
			Einreichfrist:   nil,
			Quelle:          "https://www.sfg.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("KMU und IT-Security-Anbieter für ISO27001. Verbesserung der IT- und Cybersicherheit. Laufend."),
			Active:          true,
		},
		{
			ID:           "stmk-lebens-nah",
			Name:         "SFG Lebens!Nah",
			Beschreibung: "20-30% Zuschuss für Kleinstbetriebe (Gewerbe, Handel) für Investitionen inkl. Digitalisierung.",
			Traeger:      "land",
			Bundesland:   ptr("steiermark"),
			Art:          "zuschuss",
			MaxBetrag:    30000,
			Foerderquote: ptrFloat64(0.3),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst"},
			},
			Themen:          []string{"investition", "digitalisierung"},
			Einreichfrist:   nil,
			Quelle:          "https://www.sfg.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Kleinstbetriebe im Gewerbe und Handel. Investitionen inkl. Digitalisierungsmaßnahmen."),
			Active:          true,
		},
		{
			ID:           "stmk-top-job",
			Name:         "SFG Top!Job",
			Beschreibung: "Bis 50% Zuschuss für Projekte zur Schaffung attraktiver Arbeitsplätze (New Work).",
			Traeger:      "land",
			Bundesland:   ptr("steiermark"),
			Art:          "zuschuss",
			MaxBetrag:    50000,
			Foerderquote: ptrFloat64(0.5),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"personal"},
			Einreichfrist:   nil,
			Quelle:          "https://www.sfg.at/foerderungen/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("New Work-Konzepte. Attraktive Arbeitsplatzgestaltung. Fachkräftesicherung."),
			Active:          true,
		},

		// === Oberösterreich (4 programs) ===
		{
			ID:           "ooe-innovationsassistent",
			Name:         "OÖ Innovationsassistent",
			Beschreibung: "Personalkostenzuschuss für F&E-Personal. 25-45% der Projektkosten je nach Unternehmensgröße.",
			Traeger:      "land",
			Bundesland:   ptr("oberoesterreich"),
			Art:          "zuschuss",
			MaxBetrag:    100000,
			Foerderquote: ptrFloat64(0.45),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"innovation", "personal"},
			Einreichfrist:   nil,
			Quelle:          "https://www.biz-up.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Förderung von F&E-Personal in OÖ-Unternehmen. Produkt- und Technologieinnovationen. Kooperation mit FHs/JKU möglich."),
			Active:          true,
		},
		{
			ID:           "ooe-digital-starter",
			Name:         "OÖ Digital Starter",
			Beschreibung: "Zuschuss in Staffelhöhe für Digitalisierung. Phase 1: bis €6.000, Phase 2: bis €12.000 (je ~50%).",
			Traeger:      "land",
			Bundesland:   ptr("oberoesterreich"),
			Art:          "zuschuss",
			MaxBetrag:    12000,
			Foerderquote: ptrFloat64(0.5),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"digitalisierung"},
			Einreichfrist:   nil,
			Quelle:          "https://www.biz-up.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Einführung von E-Business, Digitalisierung von Prozessen. Calls jährlich. Land OÖ/WKO."),
			Active:          true,
		},
		{
			ID:           "ooe-investitionsfoerderung",
			Name:         "OÖ Investitions- und Standortförderung",
			Beschreibung: "10-20% Zuschuss oder zinsgünstiges Landesdarlehen für Betriebsinvestitionen.",
			Traeger:      "land",
			Bundesland:   ptr("oberoesterreich"),
			Art:          "zuschuss",
			MaxBetrag:    500000,
			Foerderquote: ptrFloat64(0.2),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"investition"},
			Einreichfrist:   nil,
			Quelle:          "https://www.biz-up.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Bauliche Erweiterungen, Produktionsanlagen. An Arbeitsplatzschaffung und regionale Effekte gekoppelt. Business Upper Austria berät."),
			Active:          true,
		},
		{
			ID:           "ooe-upperwork",
			Name:         "OÖ upperWORK Qualifizierung",
			Beschreibung: "Zuschüsse zu Weiterbildungskosten für Beschäftigte in oberösterreichischen Unternehmen.",
			Traeger:      "land",
			Bundesland:   ptr("oberoesterreich"),
			Art:          "zuschuss",
			MaxBetrag:    5000,
			Foerderquote: ptrFloat64(0.5),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"personal"},
			Einreichfrist:   nil,
			Quelle:          "https://www.biz-up.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Fachkräftequalifizierung. Ergänzend zur AMS-Lehrlingsbeihilfe. Land OÖ/AMS."),
			Active:          true,
		},

		// === Salzburg (3 programs) ===
		{
			ID:           "sbg-innovationsoffensive",
			Name:         "Salzburg Innovationsoffensive",
			Beschreibung: "Co-Funding von FFG-Basisprogramm-Projekten durch Land Salzburg (ITG).",
			Traeger:      "land",
			Bundesland:   ptr("salzburg"),
			Art:          "zuschuss",
			MaxBetrag:    200000,
			Foerderquote: ptrFloat64(0.3),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"innovation"},
			Einreichfrist:   nil,
			Quelle:          "https://www.itg-salzburg.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Stärkung der Innovationskraft. Kofinanzierung zu FFG-Projekten. Beratung via Standortagentur ITG."),
			Active:          true,
		},
		{
			ID:           "sbg-wachstumsbonus",
			Name:         "Salzburg Wachstumsbonus",
			Beschreibung: "~10% Investitionszuschuss für Betriebserweiterungen in Salzburg.",
			Traeger:      "land",
			Bundesland:   ptr("salzburg"),
			Art:          "zuschuss",
			MaxBetrag:    100000,
			Foerderquote: ptrFloat64(0.1),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"investition"},
			Einreichfrist:   nil,
			Quelle:          "https://www.itg-salzburg.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Für Erweiterungsinvestitionen. Land Salzburg (ITG)."),
			Active:          true,
		},
		{
			ID:           "sbg-digital-start",
			Name:         "Salzburg Digital Start",
			Beschreibung: "Pauschal €4.000 (max. 40%) für Webshops, Online-Marketing oder Geschäftsprozesse.",
			Traeger:      "land",
			Bundesland:   ptr("salzburg"),
			Art:          "zuschuss",
			MaxBetrag:    4000,
			Foerderquote: ptrFloat64(0.4),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"digitalisierung"},
			Einreichfrist:   ptr("2025-12-31"),
			Quelle:          "https://www.itg-salzburg.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Salzburger KMU. Webshops, Online-Marketing, Geschäftsprozesse. Laufzeit bis 31.12.2025."),
			Active:          true,
		},

		// === Tirol (3 programs) ===
		{
			ID:           "tir-digitalisierungsoffensive",
			Name:         "Tirol Digitalisierungsoffensive",
			Beschreibung: "~30% Zuschuss (max. €10.000) für neue digitale Technologien, E-Commerce, Prozessdigitalisierung.",
			Traeger:      "land",
			Bundesland:   ptr("tirol"),
			Art:          "zuschuss",
			MaxBetrag:    10000,
			Foerderquote: ptrFloat64(0.3),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"digitalisierung"},
			Einreichfrist:   ptr("2025-12-31"),
			Quelle:          "https://www.standort-tirol.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Tiroler KMU. Einführung neuer digitaler Technologien. Aktion 2023-2025."),
			Active:          true,
		},
		{
			ID:           "tir-investitionsfoerderung",
			Name:         "Tirol Investitionsförderung",
			Beschreibung: "5-15% Zuschuss oder Zinszuschuss für bauliche Investitionen in peripheren Bezirken.",
			Traeger:      "land",
			Bundesland:   ptr("tirol"),
			Art:          "zuschuss",
			MaxBetrag:    200000,
			Foerderquote: ptrFloat64(0.15),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"investition"},
			Einreichfrist:   nil,
			Quelle:          "https://www.standort-tirol.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Investitionen in periphere Bezirke. Maschinen und bauliche Erweiterungen. Infrastrukturförderung für Gewerbegebiete."),
			Active:          true,
		},
		{
			ID:           "tir-exportfoerderung",
			Name:         "Tirol Exportförderung",
			Beschreibung: "Messekosten-Zuschuss bis ~€5.000 für Tiroler Unternehmen.",
			Traeger:      "land",
			Bundesland:   ptr("tirol"),
			Art:          "zuschuss",
			MaxBetrag:    5000,
			Foerderquote: ptrFloat64(0.5),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"export"},
			Einreichfrist:   nil,
			Quelle:          "https://www.standort-tirol.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Clusterförderungen über Standortagentur. F&E-Kooperationen. Messeteilnahmen."),
			Active:          true,
		},

		// === Kärnten (4 programs) ===
		{
			ID:           "ktn-digitalisierungs-impuls",
			Name:         "KWF Digitalisierungs-IMPULS",
			Beschreibung: "30% Zuschuss (max. €18.000) für Digitalprojekte ab €5.000 Invest. Kombinierbar mit KMU.Digital.",
			Traeger:      "land",
			Bundesland:   ptr("kaernten"),
			Art:          "zuschuss",
			MaxBetrag:    18000,
			Foerderquote: ptrFloat64(0.3),
			MinProjektkosten: ptrInt(5000),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"digitalisierung"},
			Einreichfrist:   nil,
			Quelle:          "https://www.kwf.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Kärntner KMU. Digitalisierung von Geschäftsprozessen, E-Commerce, IT-Security. Kombinierbar mit Bundes-KMU.Digital."),
			Active:          true,
		},
		{
			ID:           "ktn-produktion-invest",
			Name:         "KWF PRODUKTION.Invest",
			Beschreibung: "Nicht rückzahlbarer Zuschuss gestaffelt nach Größe (z.B. 7,5% für KMU, max. €500k) für Produktionsinvestitionen.",
			Traeger:      "land",
			Bundesland:   ptr("kaernten"),
			Art:          "zuschuss",
			MaxBetrag:    500000,
			Foerderquote: ptrFloat64(0.075),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel", "gross"},
			},
			Themen:          []string{"investition"},
			Einreichfrist:   nil,
			Quelle:          "https://www.kwf.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Industrieinvestitionen in Kärnten. Erweiterung oder Modernisierung von Produktionsanlagen. Calls mehrmals jährlich."),
			Active:          true,
		},
		{
			ID:           "ktn-exportoffensive",
			Name:         "KWF Exportoffensive Kärnten",
			Beschreibung: "Exportstart-Bonus: Kostenübernahme für Exportberatung, Marktrecherchen für Erstexporteure.",
			Traeger:      "land",
			Bundesland:   ptr("kaernten"),
			Art:          "zuschuss",
			MaxBetrag:    10000,
			Foerderquote: ptrFloat64(0.5),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"export"},
			Einreichfrist:   nil,
			Quelle:          "https://www.kwf.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Neu ab 2025. Für Unternehmen aller Branchen mit erstmaligen Exportplänen."),
			Active:          true,
		},
		{
			ID:           "ktn-kleinunternehmerzuschuss",
			Name:         "KWF Kleinunternehmerzuschuss",
			Beschreibung: "Einmalzuschuss bis €7.500 für Kleinstbetriebe in Kärnten.",
			Traeger:      "land",
			Bundesland:   ptr("kaernten"),
			Art:          "zuschuss",
			MaxBetrag:    7500,
			Foerderquote: ptrFloat64(0.3),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst"},
			},
			Themen:          []string{"investition", "gruendung"},
			Einreichfrist:   nil,
			Quelle:          "https://www.kwf.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Kleinstbetriebe. Einmalige Investitionsförderung. KWF."),
			Active:          true,
		},

		// === Vorarlberg (3 programs) ===
		{
			ID:           "vbg-digitalisierungszuschuss",
			Name:         "Vorarlberg Digitalisierungszuschuss",
			Beschreibung: "30% Zuschuss (bis ~€5.000) für KMU-Digitalisierungsprojekte wie ERP-Systeme, Onlineshops.",
			Traeger:      "land",
			Bundesland:   ptr("vorarlberg"),
			Art:          "zuschuss",
			MaxBetrag:    5000,
			Foerderquote: ptrFloat64(0.3),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"digitalisierung"},
			Einreichfrist:   nil,
			Quelle:          "https://www.wisto.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Neue ERP-Systeme, Onlineshops, digitale Geschäftsmodelle. Land Vorarlberg (WISTO)."),
			Active:          true,
		},
		{
			ID:           "vbg-innovationsscheck",
			Name:         "Vorarlberg Innovationsscheck",
			Beschreibung: "€7.500 Zuschuss (75% von max. €10.000) für externe F&E-Leistungen.",
			Traeger:      "land",
			Bundesland:   ptr("vorarlberg"),
			Art:          "zuschuss",
			MaxBetrag:    7500,
			Foerderquote: ptrFloat64(0.75),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"innovation"},
			Einreichfrist:   nil,
			Quelle:          "https://www.wisto.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Produktentwicklungen, Kooperationen mit FH/Uni. Land Vorarlberg (WISTO)."),
			Active:          true,
		},
		{
			ID:           "vbg-standortfoerderung",
			Name:         "Vorarlberg Standortförderung",
			Beschreibung: "~10% einmalige Zuschüsse für Betriebserweiterungen im ländlichen Raum.",
			Traeger:      "land",
			Bundesland:   ptr("vorarlberg"),
			Art:          "zuschuss",
			MaxBetrag:    50000,
			Foerderquote: ptrFloat64(0.1),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein"},
			},
			Themen:          []string{"investition"},
			Einreichfrist:   nil,
			Quelle:          "https://www.wisto.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Kleinste Betriebe (Gewerbe, Tourismus). Betriebserweiterungen, Fachkräftesicherung. Laufend über WISTO."),
			Active:          true,
		},

		// === Burgenland (3 programs) ===
		{
			ID:           "bgld-digital",
			Name:         "Burgenland.Digital",
			Beschreibung: "30% Zuschuss (max. €6.000) für Digitalisierungsprojekte aller Art. Website, Shop, ERP-Software.",
			Traeger:      "land",
			Bundesland:   ptr("burgenland"),
			Art:          "zuschuss",
			MaxBetrag:    6000,
			Foerderquote: ptrFloat64(0.3),
			MinProjektkosten: ptrInt(2000),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"digitalisierung"},
			Einreichfrist:   nil,
			Quelle:          "https://www.wirtschaftsagentur-burgenland.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("KMU im Burgenland. Website-Relaunch, Online-Shop, ERP-Software. Projektvolumen mind. €2.000 netto. Call 2024/25."),
			Active:          true,
		},
		{
			ID:           "bgld-investitionsfoerderung",
			Name:         "Burgenland Investitions- und Technologieförderung",
			Beschreibung: "10-20% Zuschuss für Betriebsinvestitionen, Maschinen, Erweiterungen. Auch Risikokapital-Beteiligungen.",
			Traeger:      "land",
			Bundesland:   ptr("burgenland"),
			Art:          "zuschuss",
			MaxBetrag:    200000,
			Foerderquote: ptrFloat64(0.2),
			Zielgruppe: Zielgruppe{
				Groessen: []string{"epu", "kleinst", "klein", "mittel"},
			},
			Themen:          []string{"investition"},
			Einreichfrist:   nil,
			Quelle:          "https://www.wirtschaftsagentur-burgenland.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Land Burgenland (FAWI). Steigerung von Wertschöpfung und Beschäftigung. Ansiedlung, Technologietransfer, erneuerbare Energie."),
			Active:          true,
		},
		{
			ID:           "bgld-gruendungsfoerderung",
			Name:         "Burgenland Gründungsförderung",
			Beschreibung: "Zuschüsse für Unternehmensgründungen im Burgenland. In Kooperation mit KHG und Bund.",
			Traeger:      "land",
			Bundesland:   ptr("burgenland"),
			Art:          "zuschuss",
			MaxBetrag:    50000,
			Foerderquote: ptrFloat64(0.3),
			Zielgruppe: Zielgruppe{
				Groessen:      []string{"epu", "kleinst"},
				MaxAlterJahre: ptrInt(3),
			},
			Themen:          []string{"gruendung"},
			Einreichfrist:   nil,
			Quelle:          "https://www.wirtschaftsagentur-burgenland.at/",
			StandDatum:      "2025-12-02",
			Detailkriterien: ptr("Kleinbetriebsförderungen. Zusammenarbeit mit KHG und Bundesförderungen. WiBAG."),
			Active:          true,
		},
	}
}

// SeedFoerderung represents a Förderung in the seed data format (matching TypeScript)
type SeedFoerderung struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Beschreibung     string     `json:"beschreibung"`
	Traeger          string     `json:"traeger"`
	Bundesland       *string    `json:"bundesland"`
	Art              string     `json:"art"`
	MaxBetrag        int        `json:"maxBetrag"`
	Foerderquote     *float64   `json:"foerderquote,omitempty"`
	MinProjektkosten *int       `json:"minProjektkosten,omitempty"`
	Zielgruppe       Zielgruppe `json:"zielgruppe"`
	Themen           []string   `json:"themen"`
	Einreichfrist    *string    `json:"einreichfrist"`
	Quelle           string     `json:"quelle"`
	StandDatum       string     `json:"standDatum"`
	Detailkriterien  *string    `json:"detailkriterien,omitempty"`
	Active           bool       `json:"active"`
}

// Zielgruppe represents the target group for a Förderung
type Zielgruppe struct {
	Groessen          []string `json:"groessen"`
	MaxAlterJahre     *int     `json:"maxAlterJahre,omitempty"`
	MinAlterJahre     *int     `json:"minAlterJahre,omitempty"`
	Branchen          []string `json:"branchen,omitempty"`
	BranchenAusschluss []string `json:"branchenAusschluss,omitempty"`
}

// Helper functions for pointers
func ptrStr(s string) *string {
	return &s
}

func ptrInt(i int) *int {
	return &i
}

func ptrFloat64(f float64) *float64 {
	return &f
}

// ptr is a generic helper for string pointers (shorthand)
func ptr(s string) *string {
	return &s
}

// ConvertSeedToFoerderung converts a SeedFoerderung to a Foerderung
// This bridges the TypeScript format to the Go types
func ConvertSeedToFoerderung(seed SeedFoerderung) *Foerderung {
	fd := &Foerderung{
		Name:        seed.Name,
		Description: &seed.Beschreibung,
		Provider:    mapTraegerToProvider(seed.Traeger),
		Topics:      seed.Themen,
		MaxAmount:   &seed.MaxBetrag,
		MinAmount:   seed.MinProjektkosten,
		Status:      StatusActive,
	}

	// Map funding type
	fd.Type = mapArtToType(seed.Art)

	// Map funding rate
	fd.FundingRateMax = seed.Foerderquote

	// Map target states (Bundesland)
	if seed.Bundesland != nil {
		fd.TargetStates = []string{*seed.Bundesland}
	} else {
		// National programs have no Bundesland = available everywhere
		fd.TargetStates = []string{}
	}

	// Map target sizes (Zielgruppe.Groessen)
	fd.TargetSizes = make([]CompanySize, 0, len(seed.Zielgruppe.Groessen))
	for _, g := range seed.Zielgruppe.Groessen {
		fd.TargetSizes = append(fd.TargetSizes, CompanySize(g))
	}

	// Map age restrictions
	fd.TargetAgeMax = seed.Zielgruppe.MaxAlterJahre
	fd.TargetAgeMin = seed.Zielgruppe.MinAlterJahre

	// Map industries
	fd.TargetIndustries = seed.Zielgruppe.Branchen
	fd.ExcludedIndustries = seed.Zielgruppe.BranchenAusschluss

	// Map requirements
	fd.Requirements = seed.Detailkriterien

	// Map URL
	fd.URL = &seed.Quelle

	// Map deadline (einreichfrist is a string in TypeScript format)
	// For now, leave it nil - would need parsing
	fd.ApplicationDeadline = nil
	if seed.Einreichfrist == nil {
		rolling := DeadlineRolling
		fd.DeadlineType = &rolling
	}

	// Set source info
	fd.Source = &seed.ID
	fd.SourceID = &seed.ID

	// Set status
	if !seed.Active {
		fd.Status = StatusClosed
	}

	return fd
}

// ConvertAllSeedData converts all seed data to Foerderung slice
func ConvertAllSeedData() []*Foerderung {
	seeds := GetSeedDataFromTypeScript()
	result := make([]*Foerderung, 0, len(seeds))
	for _, seed := range seeds {
		result = append(result, ConvertSeedToFoerderung(seed))
	}
	return result
}

// mapTraegerToProvider maps TypeScript Traeger to Go Provider constant
func mapTraegerToProvider(traeger string) string {
	switch traeger {
	case "aws":
		return ProviderAWS
	case "ffg":
		return ProviderFFG
	case "wko":
		return ProviderWKO
	case "ams":
		return ProviderAMS
	case "oekb":
		return ProviderOeKB
	case "eu":
		return ProviderEU
	default:
		// For regional providers (Wien, Steiermark, etc.), use the name directly
		return traeger
	}
}

// mapArtToType maps TypeScript Art to Go FoerderungType
func mapArtToType(art string) FoerderungType {
	switch art {
	case "zuschuss":
		return TypeZuschuss
	case "kredit":
		return TypeKredit
	case "garantie":
		return TypeGarantie
	case "beteiligung":
		return TypeKombination // Map to combination as closest match
	default:
		return TypeZuschuss
	}
}
