/// <reference types="@sveltejs/kit" />

declare global {
	namespace App {
		interface Locals {
			clientId?: string;
			tenantId?: string;
		}
		interface PageData {
			branding?: {
				company_name: string;
				logo_url?: string;
				primary_color: string;
				secondary_color?: string;
				accent_color?: string;
				welcome_message?: string;
				footer_text?: string;
			};
		}
	}
}

export {};
