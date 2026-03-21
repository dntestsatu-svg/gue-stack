import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { Button } from '@/components/ui/button'

describe('Button component', () => {
  it('renders label and classes', () => {
    const wrapper = mount(Button, {
      props: { variant: 'outline' },
      slots: { default: 'Click me' },
    })

    expect(wrapper.text()).toContain('Click me')
    expect(wrapper.attributes('class')).toContain('border')
  })
})
