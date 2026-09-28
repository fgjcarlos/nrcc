import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { I18nProvider } from '@/i18n';
import { NrccAccessBoundaryCard } from './NrccAccessBoundaryCard';
import type { User } from '@/features/auth/services/authService';

function buildUser(overrides: Partial<User> = {}): User {
  return {
    id: 'user-1',
    username: 'admin',
    role: 'admin',
    createdAt: '2026-01-01T00:00:00.000Z',
    ...overrides,
  };
}

function renderCard(props: Parameters<typeof NrccAccessBoundaryCard>[0]) {
  return render(
    <I18nProvider>
      <MemoryRouter>
        <NrccAccessBoundaryCard {...props} />
      </MemoryRouter>
    </I18nProvider>,
  );
}

describe('NrccAccessBoundaryCard', () => {
  it('renders the success palette when at least one admin exists', () => {
    renderCard({ users: [buildUser({ username: 'admin', role: 'admin' })] });
    const card = screen.getByTestId('boundary-nrcc-access');
    const chip = card.querySelector('[role="status"]');
    expect(chip?.className).toMatch(/bg-ds-success/);
  });

  it('renders the warning palette when no users exist', () => {
    renderCard({ users: [] });
    const card = screen.getByTestId('boundary-nrcc-access');
    const chip = card.querySelector('[role="status"]');
    expect(chip?.className).toMatch(/bg-ds-warning/);
  });

  it('renders the warning palette when only viewers exist', () => {
    renderCard({
      users: [buildUser({ id: 'v1', username: 'viewer', role: 'viewer' })],
    });
    const card = screen.getByTestId('boundary-nrcc-access');
    const chip = card.querySelector('[role="status"]');
    expect(chip?.className).toMatch(/bg-ds-warning/);
  });

  it('renders the warning palette when users are missing (loading state)', () => {
    renderCard({});
    const card = screen.getByTestId('boundary-nrcc-access');
    const chip = card.querySelector('[role="status"]');
    expect(chip?.className).toMatch(/bg-ds-warning/);
    // The total panel should show 0.
    expect(screen.getByTestId('boundary-nrcc-access-total')).toHaveTextContent('0');
  });

  it('exposes the admin and viewer counts in the role distribution', () => {
    renderCard({
      users: [
        buildUser({ id: 'a1', username: 'admin1', role: 'admin' }),
        buildUser({ id: 'a2', username: 'admin2', role: 'admin' }),
        buildUser({ id: 'v1', username: 'viewer1', role: 'viewer' }),
      ],
    });
    const panel = screen.getByTestId('boundary-nrcc-access-roles');
    expect(panel).toHaveTextContent(/2/);
    expect(panel).toHaveTextContent(/1/);
    expect(screen.getByTestId('boundary-nrcc-access-total')).toHaveTextContent('3');
  });

  it('deep-links to /settings/users', () => {
    renderCard({ users: [buildUser()] });
    const link = screen.getByTestId('boundary-nrcc-access-link');
    expect(link).toHaveAttribute('href', '/settings/users');
  });

  it('renders the success chip with the user count when at least one user exists', () => {
    renderCard({
      users: [buildUser({ username: 'admin', role: 'admin' })],
    });
    const card = screen.getByTestId('boundary-nrcc-access');
    const chip = card.querySelector('[role="status"]');
    expect(chip?.textContent).toMatch(/1/);
  });
});
