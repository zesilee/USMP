import { test, expect } from '@playwright/test'

test('interfaces grid page renders backend-driven grid', async ({ page }) => {
  await page.route('**/api/v1/ui-schema/devices/*/interfaces', async route => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        code: 0,
        message: 'UI schema retrieved',
        data: {
          schemaVersion: 'interfaces:v1',
          module: 'huawei-ifm',
          targetPath: '/ifm:ifm/ifm:interfaces',
          capabilitySource: 'module-set',
          layout: { type: 'grid', columns: 12, gap: 'md' },
          sections: [{ id: 'interfaces', title: '接口配置', widgets: ['interfaces-table'] }],
          widgets: [{
            id: 'interfaces-table',
            type: 'table',
            label: '接口列表',
            rowKey: 'name',
            grid: { span: 12 },
            columns: [{ id: 'name', type: 'text', label: '接口名称' }]
          }],
          values: { 'interfaces-table': [{ name: 'GigabitEthernet0/0/1' }] }
        }
      })
    })
  })

  await page.goto('/config/interface')

  await expect(page.getByRole('heading', { name: '接口配置' })).toBeVisible()
  await expect(page.getByRole('heading', { name: '接口列表' })).toBeVisible()
  await expect(page.getByText('GigabitEthernet0/0/1')).toBeVisible()
})
