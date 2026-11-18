import { useState } from 'react'
import { describe, expect, it } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { TabPanel, Tabs } from './Tabs'

const ITEMS = [
  { id: 'one', label: 'One' },
  { id: 'two', label: 'Two' },
  { id: 'three', label: 'Three' },
]

function Harness() {
  const [selected, setSelected] = useState('one')
  return (
    <>
      <Tabs label="Sections" idBase="demo" items={ITEMS} selected={selected} onSelect={setSelected} />
      <TabPanel idBase="demo" selected={selected}>{`Panel ${selected}`}</TabPanel>
    </>
  )
}

describe('Tabs', () => {
  it('renders a labelled tablist wired to its panel with a roving tabindex', () => {
    render(<Harness />)
    const tabs = within(screen.getByRole('tablist', { name: 'Sections' })).getAllByRole('tab')
    expect(tabs.map((tab) => tab.textContent)).toEqual(['One', 'Two', 'Three'])
    expect(tabs[0]).toHaveAttribute('aria-selected', 'true')
    expect(tabs[0]).toHaveAttribute('tabindex', '0')
    expect(tabs[1]).toHaveAttribute('aria-selected', 'false')
    expect(tabs[1]).toHaveAttribute('tabindex', '-1')
    const panel = screen.getByRole('tabpanel', { name: 'One' })
    expect(tabs[0]).toHaveAttribute('aria-controls', panel.id)
    expect(panel).toHaveTextContent('Panel one')
  })

  it('selects on click and moves with Left/Right/Home/End, wrapping at the ends', async () => {
    const user = userEvent.setup()
    render(<Harness />)
    const tabs = screen.getAllByRole('tab')
    await user.click(tabs[1])
    expect(tabs[1]).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByRole('tabpanel', { name: 'Two' })).toHaveTextContent('Panel two')

    await user.keyboard('{ArrowRight}')
    expect(tabs[2]).toHaveFocus()
    await user.keyboard('{ArrowRight}')
    expect(tabs[0]).toHaveFocus()
    expect(tabs[0]).toHaveAttribute('aria-selected', 'true')
    await user.keyboard('{ArrowLeft}')
    expect(tabs[2]).toHaveFocus()
    await user.keyboard('{Home}')
    expect(tabs[0]).toHaveFocus()
    await user.keyboard('{End}')
    expect(tabs[2]).toHaveFocus()
    expect(tabs[2]).toHaveAttribute('aria-selected', 'true')
  })
})
