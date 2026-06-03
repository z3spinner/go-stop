import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import PrivacyModal from './PrivacyModal.svelte';

describe('PrivacyModal', () => {
	it('renders the privacy heading and a close button', async () => {
		const onclose = vi.fn();
		render(PrivacyModal, { props: { onclose } });
		expect(screen.getByText(/Confidentialité|Privacy/)).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /Fermer|Close/ })).toBeInTheDocument();
	});
});
