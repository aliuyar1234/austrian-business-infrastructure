/**
 * Type definitions for Förderungsradar API responses
 */

// ==================
// Base Types
// ==================

export type CompanySize = 'all' | 'startup' | 'mikro' | 'klein' | 'mittel' | 'kmu' | 'gross';
export type FoerderungType = 'zuschuss' | 'kredit' | 'garantie' | 'beratung' | 'kombination';
export type FoerderungStatus = 'active' | 'inactive' | 'draft' | 'expired';
export type AntragStatus = 'planned' | 'drafting' | 'submitted' | 'in_review' | 'approved' | 'rejected' | 'withdrawn';
export type DigestMode = 'immediate' | 'daily' | 'weekly';

// ==================
// Förderung Types
// ==================

export interface Foerderung {
	id: string;
	tenant_id?: string;
	name: string;
	provider: string;
	type: FoerderungType;
	description?: string;
	requirements?: string;
	url?: string;
	min_amount?: number;
	max_amount?: number;
	funding_rate_min?: number;
	funding_rate_max?: number;
	target_states: string[];
	target_sizes: CompanySize[];
	target_industries?: string[];
	excluded_industries?: string[];
	target_age_min?: number;
	target_age_max?: number;
	topics: string[];
	application_deadline?: string;
	status: FoerderungStatus;
	combinable_with?: string[];
	not_combinable_with?: string[];
	created_at: string;
	updated_at: string;
}

export interface FoerderungenListResponse {
	foerderungen: Foerderung[];
	total?: number;
}

export interface FoerderungDetailResponse extends Foerderung {}

// ==================
// Profile Types
// ==================

export interface Unternehmensprofil {
	id: string;
	tenant_id: string;
	name: string;
	company_name: string;
	legal_form?: string;
	state: string;
	district?: string;
	employees_count?: number;
	annual_revenue?: number;
	balance_total?: number;
	is_startup: boolean;
	is_kmu: boolean;
	founded_year?: number;
	onace_codes?: string[];
	project_description?: string;
	project_topics?: string[];
	investment_amount?: number;
	industry?: string;
	linked_account_id?: string;
	created_at: string;
	updated_at: string;
}

export interface ProfileListResponse {
	profiles: Unternehmensprofil[];
	total?: number;
}

export interface ProfileCreateResponse extends Unternehmensprofil {}

// ==================
// Search Types
// ==================

export interface LLMEligibilityResult {
	eligible: boolean;
	confidence: 'high' | 'medium' | 'low';
	matched_criteria?: string[];
	implicit_matches?: string[];
	concerns?: string[];
	estimated_amount?: number;
	next_steps?: string[];
	insider_tip?: string;
	combination_hint?: string;
}

export interface FoerderungsMatch {
	foerderung_id: string;
	foerderung_name: string;
	provider: string;
	rule_score: number;
	llm_score: number;
	total_score: number;
	llm_result?: LLMEligibilityResult;
}

export interface SearchResponse {
	search_id: string;
	total_checked: number;
	total_matches: number;
	matches: FoerderungsMatch[];
	llm_tokens_used: number;
	llm_cost_cents: number;
	duration: number;
	llm_fallback: boolean;
}

// ==================
// Antrag Types
// ==================

export interface Attachment {
	name: string;
	type: string;
	url: string;
	uploaded_at: string;
}

export interface TimelineEntry {
	date: string;
	status: string;
	description: string;
}

export interface Antrag {
	id: string;
	profile_id: string;
	foerderung_id: string;
	status: AntragStatus;
	internal_reference?: string;
	submitted_at?: string;
	requested_amount?: number;
	approved_amount?: number;
	decision_date?: string;
	decision_notes?: string;
	attachments: Attachment[];
	timeline: TimelineEntry[];
	notes?: string;
	created_at: string;
	updated_at: string;
}

export interface AntragStats {
	total: number;
	planned: number;
	drafting: number;
	submitted: number;
	in_review: number;
	in_progress?: number; // Alias for drafting + submitted + in_review
	approved: number;
	rejected: number;
	total_requested: number;
	total_approved: number;
	success_rate: number;
}

export interface AntraegeListResponse {
	antraege: Antrag[];
	total?: number;
}

export interface AntragDetailResponse extends Antrag {}

// ==================
// Monitor Types
// ==================

export interface Monitor {
	id: string;
	tenant_id: string;
	profile_id: string;
	threshold: number;
	digest_mode: DigestMode;
	is_active: boolean;
	last_checked_at?: string;
	created_at: string;
	updated_at: string;
}

export interface Notification {
	id: string;
	monitor_id: string;
	foerderung_id: string;
	foerderung_name: string;
	match_score: number;
	message: string;
	is_viewed: boolean;
	is_dismissed: boolean;
	viewed_at?: string;
	dismissed_at?: string;
	created_at: string;
}

export interface MonitorListResponse {
	monitors: Monitor[];
}

export interface NotificationListResponse {
	notifications: Notification[];
}

// ==================
// Combination Types
// ==================

export interface CombinableProgram {
	foerderung_id: string;
	foerderung_name: string;
	reason: string;
}

export interface CombinationsResponse {
	explicit: CombinableProgram[];
	inferred: CombinableProgram[];
}
