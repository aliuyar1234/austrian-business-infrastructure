export { cn } from './cn';

/**
 * Format a date for display
 */
export function formatDate(date: Date | string, options?: Intl.DateTimeFormatOptions): string {
	const d = typeof date === 'string' ? new Date(date) : date;
	return d.toLocaleDateString('de-AT', {
		year: 'numeric',
		month: '2-digit',
		day: '2-digit',
		...options
	});
}

/**
 * Format a date with time
 */
export function formatDateTime(date: Date | string): string {
	const d = typeof date === 'string' ? new Date(date) : date;
	return d.toLocaleString('de-AT', {
		year: 'numeric',
		month: '2-digit',
		day: '2-digit',
		hour: '2-digit',
		minute: '2-digit'
	});
}

/**
 * Format currency in EUR
 * @param amount - Amount in the currency's base unit (e.g., EUR, not cents)
 * @param currency - Currency code (default: EUR)
 * @param options - Additional Intl.NumberFormat options
 */
export function formatCurrency(
	amount: number,
	currency = 'EUR',
	options?: Partial<Intl.NumberFormatOptions>
): string {
	return new Intl.NumberFormat('de-AT', {
		style: 'currency',
		currency,
		...options
	}).format(amount);
}

/**
 * Format currency from cents (for APIs that store amounts in cents)
 * @param amountInCents - Amount in cents
 * @param currency - Currency code (default: EUR)
 */
export function formatCurrencyFromCents(amountInCents: number, currency = 'EUR'): string {
	return formatCurrency(amountInCents / 100, currency);
}

/**
 * Format a number with locale
 */
export function formatNumber(value: number): string {
	return new Intl.NumberFormat('de-AT').format(value);
}

/**
 * Debounce function for search inputs
 */
export function debounce<T extends (...args: unknown[]) => unknown>(
	fn: T,
	delay: number
): (...args: Parameters<T>) => void {
	let timeoutId: ReturnType<typeof setTimeout>;
	return (...args: Parameters<T>) => {
		clearTimeout(timeoutId);
		timeoutId = setTimeout(() => fn(...args), delay);
	};
}

/**
 * Generate initials from a name
 */
export function getInitials(name: string): string {
	return name
		.split(' ')
		.map((part) => part[0])
		.join('')
		.toUpperCase()
		.slice(0, 2);
}

/**
 * Truncate text with ellipsis
 */
export function truncate(text: string, maxLength: number): string {
	if (text.length <= maxLength) return text;
	return text.slice(0, maxLength - 3) + '...';
}

/**
 * Sleep utility for animations/delays
 */
export function sleep(ms: number): Promise<void> {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Check if we're in a browser environment
 */
export const isBrowser = typeof window !== 'undefined';

/**
 * Copy text to clipboard
 */
export async function copyToClipboard(text: string): Promise<boolean> {
	if (!isBrowser) return false;
	try {
		await navigator.clipboard.writeText(text);
		return true;
	} catch {
		return false;
	}
}
