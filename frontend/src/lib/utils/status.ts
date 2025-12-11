/**
 * Shared status helpers for consistent status display across the application
 */

export type StatusVariant = 'default' | 'success' | 'warning' | 'error' | 'info';

// Status type definitions for each domain
export type AccountStatus = 'active' | 'syncing' | 'error' | 'pending';
export type UVAStatus = 'draft' | 'submitted' | 'accepted' | 'rejected';
export type InvoiceStatus = 'draft' | 'generated' | 'sent';
export type DocumentStatus = 'unread' | 'read' | 'archived';
export type SignatureStatus = 'pending' | 'in_progress' | 'completed' | 'expired' | 'cancelled';
export type PaymentStatus = 'draft' | 'generated' | 'sent' | 'executed';
export type CompanyStatus = 'active' | 'dissolved' | 'liquidating' | 'unknown';

type StatusType = 'account' | 'uva' | 'invoice' | 'document' | 'signature' | 'payment' | 'company';

type StatusMap<T extends string> = Record<T, string>;
type VariantMap<T extends string> = Record<T, StatusVariant>;

// Status labels by type
const accountLabels: StatusMap<AccountStatus> = {
	active: 'Active',
	syncing: 'Syncing...',
	error: 'Error',
	pending: 'Pending'
};

const uvaLabels: StatusMap<UVAStatus> = {
	draft: 'Draft',
	submitted: 'Submitted',
	accepted: 'Accepted',
	rejected: 'Rejected'
};

const invoiceLabels: StatusMap<InvoiceStatus> = {
	draft: 'Draft',
	generated: 'Generated',
	sent: 'Sent'
};

const documentLabels: StatusMap<DocumentStatus> = {
	unread: 'Unread',
	read: 'Read',
	archived: 'Archived'
};

const signatureLabels: StatusMap<SignatureStatus> = {
	pending: 'Ausstehend',
	in_progress: 'In Bearbeitung',
	completed: 'Abgeschlossen',
	expired: 'Abgelaufen',
	cancelled: 'Storniert'
};

const paymentLabels: StatusMap<PaymentStatus> = {
	draft: 'Draft',
	generated: 'Generated',
	sent: 'Sent',
	executed: 'Executed'
};

const companyLabels: StatusMap<CompanyStatus> = {
	active: 'Active',
	dissolved: 'Dissolved',
	liquidating: 'In Liquidation',
	unknown: 'Unknown'
};

// Status variants by type
const accountVariants: VariantMap<AccountStatus> = {
	active: 'success',
	syncing: 'info',
	error: 'error',
	pending: 'warning'
};

const uvaVariants: VariantMap<UVAStatus> = {
	draft: 'default',
	submitted: 'warning',
	accepted: 'success',
	rejected: 'error'
};

const invoiceVariants: VariantMap<InvoiceStatus> = {
	draft: 'default',
	generated: 'warning',
	sent: 'success'
};

const documentVariants: VariantMap<DocumentStatus> = {
	unread: 'info',
	read: 'default',
	archived: 'default'
};

const signatureVariants: VariantMap<SignatureStatus> = {
	pending: 'warning',
	in_progress: 'info',
	completed: 'success',
	expired: 'error',
	cancelled: 'default'
};

const paymentVariants: VariantMap<PaymentStatus> = {
	draft: 'default',
	generated: 'warning',
	sent: 'info',
	executed: 'success'
};

const companyVariants: VariantMap<CompanyStatus> = {
	active: 'success',
	dissolved: 'error',
	liquidating: 'warning',
	unknown: 'default'
};

const labelMaps: Record<StatusType, StatusMap<string>> = {
	account: accountLabels,
	uva: uvaLabels,
	invoice: invoiceLabels,
	document: documentLabels,
	signature: signatureLabels,
	payment: paymentLabels,
	company: companyLabels
};

const variantMaps: Record<StatusType, VariantMap<string>> = {
	account: accountVariants,
	uva: uvaVariants,
	invoice: invoiceVariants,
	document: documentVariants,
	signature: signatureVariants,
	payment: paymentVariants,
	company: companyVariants
};

/**
 * Get a human-readable label for a status value
 * @param status - The status value
 * @param type - The type of entity (account, uva, invoice, document, signature, payment, company)
 * @returns The human-readable label
 */
export function getStatusLabel(status: string, type: StatusType): string {
	const labels = labelMaps[type];
	return labels?.[status] ?? status;
}

/**
 * Get the badge variant for a status value
 * @param status - The status value
 * @param type - The type of entity (account, uva, invoice, document, signature, payment, company)
 * @returns The badge variant
 */
export function getStatusVariant(status: string, type: StatusType): StatusVariant {
	const variants = variantMaps[type];
	return variants?.[status] ?? 'default';
}
