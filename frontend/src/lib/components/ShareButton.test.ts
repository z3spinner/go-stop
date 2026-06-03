import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import ShareButton from './ShareButton.svelte';

afterEach(() => vi.unstubAllGlobals());

describe('ShareButton', () => {
	it('uses the native share sheet when available', async () => {
		const shareSpy = vi.fn().mockResolvedValue(undefined);
		vi.stubGlobal('navigator', { share: shareSpy });

		const { container } = render(ShareButton, {
			props: { title: 'Saillans → Crest', text: 'hi', url: 'https://x/rides/1' }
		});
		await fireEvent.click(container.querySelector('.btn-share')!);

		expect(shareSpy).toHaveBeenCalledWith({ title: 'Saillans → Crest', text: 'hi', url: 'https://x/rides/1' });
	});

	it('falls back to copying the link and confirms inline', async () => {
		const writeText = vi.fn().mockResolvedValue(undefined);
		vi.stubGlobal('navigator', { clipboard: { writeText } }); // no Web Share API

		const { container } = render(ShareButton, { props: { title: 'T', url: 'https://x/rides/2' } });
		const btn = container.querySelector('.btn-share') as HTMLElement;
		await fireEvent.click(btn);

		expect(writeText).toHaveBeenCalledWith('https://x/rides/2');
		// icon button confirms via its title attribute (no text label)
		await vi.waitFor(() => expect(btn.getAttribute('title')).toMatch(/copié|copied/i));
	});
});
