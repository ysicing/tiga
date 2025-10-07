import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { EmptyState } from '../empty-state'

describe('EmptyState', () => {
  it('renders message and action text', () => {
    render(
      <EmptyState
        message="No instances found"
        actionText="Create Instance"
        onAction={vi.fn()}
      />
    )

    expect(screen.getByText('No instances found')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /create instance/i })).toBeInTheDocument()
  })

  it('calls onAction when button is clicked', () => {
    const handleAction = vi.fn()
    render(
      <EmptyState
        message="No data available"
        actionText="Add Data"
        onAction={handleAction}
      />
    )

    const button = screen.getByRole('button', { name: /add data/i })
    fireEvent.click(button)

    expect(handleAction).toHaveBeenCalledTimes(1)
  })

  it('renders custom icon when provided', () => {
    const CustomIcon = () => <svg data-testid="custom-icon" />

    render(
      <EmptyState
        message="Empty state"
        actionText="Action"
        onAction={vi.fn()}
        icon={<CustomIcon />}
      />
    )

    expect(screen.getByTestId('custom-icon')).toBeInTheDocument()
  })

  it('renders without action button when onAction is not provided', () => {
    render(
      <EmptyState
        message="Read-only empty state"
      />
    )

    expect(screen.getByText('Read-only empty state')).toBeInTheDocument()
    expect(screen.queryByRole('button')).toBeNull()
  })

  it('supports keyboard navigation', () => {
    const handleAction = vi.fn()
    render(
      <EmptyState
        message="Test message"
        actionText="Test action"
        onAction={handleAction}
      />
    )

    const button = screen.getByRole('button')
    button.focus()
    fireEvent.keyDown(button, { key: 'Enter' })

    expect(handleAction).toHaveBeenCalledTimes(1)
  })
})
