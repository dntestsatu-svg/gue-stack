import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import DateRangePicker from '@/components/DateRangePicker.vue'

describe('DateRangePicker', () => {
  it('renders formatted trigger labels from the provided model value', async () => {
    const wrapper = mount(DateRangePicker, {
      props: {
        modelValue: {
          from: '2026-03-10',
          to: '2026-03-15',
        },
      },
    })

    expect(wrapper.text()).toContain('10 Mar 2026')
    expect(wrapper.text()).toContain('15 Mar 2026')
  })

  it('emits an empty range when the clear preset is clicked', async () => {
    const wrapper = mount(DateRangePicker, {
      props: {
        modelValue: {
          from: '2026-03-10',
          to: '2026-03-15',
        },
      },
    })

    await wrapper.get('button:nth-of-type(4)').trigger('click')

    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual({
      from: '',
      to: '',
    })
  })

  it('emits a valid range when the last 7 days preset is clicked', async () => {
    const wrapper = mount(DateRangePicker, {
      props: {
        modelValue: {
          from: '',
          to: '',
        },
      },
    })

    await wrapper.get('button:nth-of-type(2)').trigger('click')

    const emitted = wrapper.emitted('update:modelValue')?.at(-1)?.[0] as { from: string, to: string }
    expect(emitted.from).toMatch(/^\d{4}-\d{2}-\d{2}$/)
    expect(emitted.to).toMatch(/^\d{4}-\d{2}-\d{2}$/)
    expect(emitted.from <= emitted.to).toBe(true)
  })
})
