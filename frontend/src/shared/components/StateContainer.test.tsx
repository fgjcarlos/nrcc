import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { StateContainer } from './StateContainer';

describe('StateContainer', () => {
  it('renders loading slot when isLoading is true', () => {
    render(
      <StateContainer isLoading={true} isError={false} isEmpty={false}>
        <div>Children</div>
      </StateContainer>
    );

    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  it('renders custom loading slot when provided', () => {
    render(
      <StateContainer
        isLoading={true}
        isError={false}
        isEmpty={false}
        loadingSlot={<div>Custom loading</div>}
      >
        <div>Children</div>
      </StateContainer>
    );

    expect(screen.getByText('Custom loading')).toBeInTheDocument();
    expect(screen.queryByText('Children')).not.toBeInTheDocument();
  });

  it('renders error slot when isError is true', () => {
    render(
      <StateContainer isLoading={false} isError={true} isEmpty={false}>
        <div>Children</div>
      </StateContainer>
    );

    expect(screen.getByText('An error occurred')).toBeInTheDocument();
    expect(screen.queryByText('Children')).not.toBeInTheDocument();
  });

  it('renders custom error slot when provided', () => {
    render(
      <StateContainer
        isLoading={false}
        isError={true}
        isEmpty={false}
        errorSlot={<div>Custom error</div>}
      >
        <div>Children</div>
      </StateContainer>
    );

    expect(screen.getByText('Custom error')).toBeInTheDocument();
  });

  it('renders empty slot when isEmpty is true', () => {
    render(
      <StateContainer isLoading={false} isError={false} isEmpty={true}>
        <div>Children</div>
      </StateContainer>
    );

    expect(screen.getByText('No items yet')).toBeInTheDocument();
    expect(screen.queryByText('Children')).not.toBeInTheDocument();
  });

  it('renders custom empty slot when provided', () => {
    render(
      <StateContainer
        isLoading={false}
        isError={false}
        isEmpty={true}
        emptySlot={<div>Custom empty</div>}
      >
        <div>Children</div>
      </StateContainer>
    );

    expect(screen.getByText('Custom empty')).toBeInTheDocument();
  });

  it('renders children when all state flags are false', () => {
    render(
      <StateContainer isLoading={false} isError={false} isEmpty={false}>
        <div>Children Content</div>
      </StateContainer>
    );

    expect(screen.getByText('Children Content')).toBeInTheDocument();
    expect(screen.queryByText('Loading...')).not.toBeInTheDocument();
    expect(screen.queryByText('An error occurred')).not.toBeInTheDocument();
    expect(screen.queryByText('No items yet')).not.toBeInTheDocument();
  });

  it('prioritizes loading over error', () => {
    render(
      <StateContainer isLoading={true} isError={true} isEmpty={false}>
        <div>Children</div>
      </StateContainer>
    );

    expect(screen.getByText('Loading...')).toBeInTheDocument();
    expect(screen.queryByText('An error occurred')).not.toBeInTheDocument();
  });

  it('prioritizes error over empty', () => {
    render(
      <StateContainer isLoading={false} isError={true} isEmpty={true}>
        <div>Children</div>
      </StateContainer>
    );

    expect(screen.getByText('An error occurred')).toBeInTheDocument();
    expect(screen.queryByText('No items yet')).not.toBeInTheDocument();
  });

  it('prioritizes empty over children', () => {
    render(
      <StateContainer isLoading={false} isError={false} isEmpty={true}>
        <div>Children</div>
      </StateContainer>
    );

    expect(screen.getByText('No items yet')).toBeInTheDocument();
    expect(screen.queryByText('Children')).not.toBeInTheDocument();
  });
});
