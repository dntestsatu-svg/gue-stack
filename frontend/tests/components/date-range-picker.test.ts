import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import DateRangePicker from '@/components/DateRangePicker.vue'

describe('DateRangePicker', () => {
  it('normalizes the range when the from date moves past the to date', async () => {
    const wrapper = mount(DateRangePicker, {
      props: {
        modelValue: {
          from: '2026-03-10',
          to: '2026-03-15',
        },
      },
    })

    const inputs = wrapper.findAll('input[type="date"]')
    await inputs[0].setValue('2026-03-20')

    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual({
      from: '2026-03-20',
      to: '2026-03-20',
    })
  })

  it('clears both dates when the clear preset is clicked', async () => {
    const wrapper = mount(DateRangePicker, {
      props: {
        modelValue: {
          from: '2026-03-10',
          to: '2026-03-15',
        },
      },
    })

    await wrapper.get('button:last-of-type').trigger('click')

    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual({
      from: '',
      to: '',
    })
  })
})
