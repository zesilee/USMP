// zustand 测试隔离（替代 Pinia 时代的 setActivePinia(createPinia())）：
// 各 store 为模块级单例，用例间以「模块加载时初始快照」整体重置
// （setState(init, true) 连同 actions 一并恢复）。locale 依赖 localStorage
// 的初始化走其 __resetForTest 重算，不在此列。
import { useDeviceStore } from '../../src/stores/device'
import { useMenuStore } from '../../src/stores/menu'
import { useFreshnessStore } from '../../src/stores/freshness'
import { useChangesetStore } from '../../src/stores/changeset'

const snapshots: Array<[{ setState: (s: any, replace: true) => void }, unknown]> = [
  [useDeviceStore, useDeviceStore.getState()],
  [useMenuStore, useMenuStore.getState()],
  [useFreshnessStore, useFreshnessStore.getState()],
  [useChangesetStore, useChangesetStore.getState()],
]

export function resetStores(): void {
  for (const [store, init] of snapshots) store.setState(init, true)
}
