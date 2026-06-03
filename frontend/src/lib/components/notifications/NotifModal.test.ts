import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import NotifModal from './NotifModal.svelte';
import { notifModalState } from '$lib/notifModal';

beforeEach(() => notifModalState.set(null));

describe('NotifModal', () => {
	it('is closed when state is null', () => {
		const { container } = render(NotifModal);
		expect(container.querySelector('.modal-overlay')).toBeNull();
	});
	it('default state shows enable + skip buttons', async () => {
		render(NotifModal);
		notifModalState.set('default');
		await Promise.resolve();
		expect(await screen.findByText(/Activer les notifications|Enable notifications/)).toBeInTheDocument();
	});
});
