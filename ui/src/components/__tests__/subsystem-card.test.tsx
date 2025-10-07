import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { SubsystemCard } from '../subsystem-card'

describe('SubsystemCard', () => {
  it('renders with correct type and stats', () => {
    render(
      <SubsystemCard
        type="mysql"
        count={10}
        running={8}
        stopped={1}
        error={1}
        onClick={vi.fn()}
        hasPermission={true}
      />
    )

    expect(screen.getByText('mysql', { exact: false })).toBeInTheDocument()
    expect(screen.getByText(/10/)).toBeInTheDocument()
    expect(screen.getByText(/8.*running/i)).toBeInTheDocument()
  })

  it('renders empty state when count is zero', () => {
    render(
      <SubsystemCard
        type="redis"
        count={0}
        running={0}
        stopped={0}
        error={0}
        onClick={vi.fn()}
        hasPermission={true}
      />
    )

    expect(screen.getByText(/no instances.*create/i)).toBeInTheDocument()
  })

  it('calls onClick handler when clicked', () => {
    const handleClick = vi.fn()
    render(
      <SubsystemCard
        type="postgresql"
        count={5}
        running={5}
        stopped={0}
        error={0}
        onClick={handleClick}
        hasPermission={true}
      />
    )

    const card = screen.getByRole('button')
    fireEvent.click(card)

    expect(handleClick).toHaveBeenCalledTimes(1)
  })

  it('does not render when user has no permission', () => {
    const { container } = render(
      <SubsystemCard
        type="minio"
        count={3}
        running={3}
        stopped={0}
        error={0}
        onClick={vi.fn()}
        hasPermission={false}
      />
    )

    expect(container.firstChild).toBeNull()
  })

  it('displays error status with correct styling', () => {
    render(
      <SubsystemCard
        type="docker"
        count={10}
        running={5}
        stopped={3}
        error={2}
        onClick={vi.fn()}
        hasPermission={true}
      />
    )

    expect(screen.getByText(/2.*error/i)).toBeInTheDocument()
  })

  it('is keyboard accessible', () => {
    const handleClick = vi.fn()
    render(
      <SubsystemCard
        type="kubernetes"
        count={3}
        running={3}
        stopped={0}
        error={0}
        onClick={handleClick}
        hasPermission={true}
      />
    )

    const card = screen.getByRole('button')
    card.focus()

    fireEvent.keyDown(card, { key: 'Enter' })
    expect(handleClick).toHaveBeenCalledTimes(1)
  })
})
