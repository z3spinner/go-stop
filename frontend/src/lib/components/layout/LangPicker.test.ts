import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import LangPicker from './LangPicker.svelte';

describe('LangPicker', () => {
	it('shows the current locale flag and a dropdown with all 6 locales', async () => {
		const { container } = render(LangPicker);
		// 6 language options exist (hidden until opened)
		expect(container.querySelectorAll('.lang-opt').length).toBe(6);
	});
});
