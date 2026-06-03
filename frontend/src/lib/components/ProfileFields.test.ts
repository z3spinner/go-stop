import { describe, it, expect } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import ProfileFields from './ProfileFields.svelte';

describe('ProfileFields', () => {
	it('shows the inputs when no profile is set', () => {
		const { container } = render(ProfileFields, { props: { name: '', phone: '', nameField: 'driver_name' } });
		expect(container.querySelector('input[name=driver_name]')).toBeTruthy();
		expect(container.querySelector('.profile-summary')).toBeNull();
	});

	it('hides the inputs and shows a summary when a profile is already filled', () => {
		const { container } = render(ProfileFields, { props: { name: 'Alice', phone: '0611000001', nameField: 'driver_name' } });
		expect(container.querySelector('input[name=driver_name]')).toBeNull();
		const summary = container.querySelector('.profile-summary');
		expect(summary?.textContent).toContain('Alice');
		expect(summary?.textContent).toContain('0611000001');
	});

	it('"Change" reveals the inputs pre-filled from the profile', async () => {
		const { container } = render(ProfileFields, { props: { name: 'Alice', phone: '0611000001', nameField: 'searcher_name' } });
		await fireEvent.click(container.querySelector('.btn-edit-contact')!);
		expect((container.querySelector('input[name=searcher_name]') as HTMLInputElement).value).toBe('Alice');
		expect((container.querySelector('input[name=phone]') as HTMLInputElement).value).toBe('0611000001');
	});
});
