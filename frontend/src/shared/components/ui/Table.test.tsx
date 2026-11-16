import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Table } from './Table'

function renderTable(stacked?: boolean) {
  return render(
    <Table caption="Environment variables" stacked={stacked}>
      <thead>
        <tr><th>Key</th><th>Size</th></tr>
      </thead>
      <tbody>
        <tr><td data-label="Key">PORT</td><td data-label="Size">1 024</td></tr>
      </tbody>
    </Table>
  )
}

describe('Table', () => {
  it('renders a native table on the shared ds-table contract with an accessible caption', () => {
    renderTable()
    const table = screen.getByRole('table', { name: 'Environment variables' })
    expect(table).toHaveClass('ds-table')
    expect(table.parentElement).toHaveClass('ds-table-wrap')
    expect(table.parentElement).not.toHaveAttribute('data-stacked')
    expect(screen.getByRole('columnheader', { name: 'Key' })).toBeInTheDocument()
    expect(screen.getByRole('cell', { name: 'PORT' })).toHaveAttribute('data-label', 'Key')
  })

  it('marks the wrapper for the labelled-card layout below sm when stacked', () => {
    renderTable(true)
    expect(screen.getByRole('table').parentElement).toHaveAttribute('data-stacked')
  })
})
