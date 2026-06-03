import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import RequestCard from './RequestCard.svelte';

const base = { id: 'int7', ride_id: 'r1', driver_name: 'Alice', origin: 'Saillans', destination: 'Crest', departure_at: '2030-06-15T08:00:00Z' };

describe('RequestCard', () => {
	it('pending shows the waiting label, no contact link', () => {
		const { container } = render(RequestCard, { props: { interest: { ...base, status: 'pending' } } });
		expect(container.querySelector('.btn-contact-link')).toBeNull();
	});
	it('accepted shows a contact link to /interests/<id>', () => {
		const { container } = render(RequestCard, { props: { interest: { ...base, status: 'accepted' } } });
		const a = container.querySelector('.btn-contact-link') as HTMLAnchorElement;
		expect(a.getAttribute('href')).toBe('/interests/int7');
	});
});
