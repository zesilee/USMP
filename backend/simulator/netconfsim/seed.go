package netconfsim

// DemoSeedConfig 是 standalone 模拟网元的默认运行配置种子：华为 IFM 5 条接口
// （3 条 main-interface/200GE/up + 2 条 sub-interface/Vlanif/down，parent-name
// 指向对应主接口），供 staging 演示与 E2E 冒烟断言。枚举值为设备侧数字形态
// （class: 1=main/2=sub；type: 93=200GE/16=Vlanif；admin-status: 2=up/1=down；
// link-protocol: 1=ethernet），与 ParseHuaweiIfmInterfacesXML 回读解析对齐。
const DemoSeedConfig = `<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm">
    <interfaces>
      <interface>
        <name>200GE0/1/0</name>
        <class>1</class>
        <type>93</type>
        <number>0/1/0</number>
        <admin-status>2</admin-status>
        <link-protocol>1</link-protocol>
        <mtu>9216</mtu>
        <description>uplink to core-1</description>
      </interface>
      <interface>
        <name>200GE0/1/1</name>
        <class>1</class>
        <type>93</type>
        <number>0/1/1</number>
        <admin-status>2</admin-status>
        <link-protocol>1</link-protocol>
        <mtu>9216</mtu>
        <description>uplink to core-2</description>
      </interface>
      <interface>
        <name>200GE0/1/2</name>
        <class>1</class>
        <type>93</type>
        <number>0/1/2</number>
        <admin-status>2</admin-status>
        <link-protocol>1</link-protocol>
        <mtu>9216</mtu>
        <description>spare uplink</description>
      </interface>
      <interface>
        <name>200GE0/1/0.1</name>
        <class>2</class>
        <type>16</type>
        <parent-name>200GE0/1/0</parent-name>
        <admin-status>1</admin-status>
        <description>tenant-a sub-interface</description>
      </interface>
      <interface>
        <name>200GE0/1/1.1</name>
        <class>2</class>
        <type>16</type>
        <parent-name>200GE0/1/1</parent-name>
        <admin-status>1</admin-status>
        <description>tenant-b sub-interface</description>
      </interface>
    </interfaces>
</ifm>`

// DemoStateSeed 是 standalone 模拟网元的默认 config-false 状态种子（NS-08）：
// 与 DemoSeedConfig 的 5 条接口按 name 键对齐，每条一个 <dynamic> 状态容器，
// 供 <get> 合并回读点亮前端只读字段。枚举为设备侧数字形态（oper/link/
// physical-status: 2=up/1=down，与 admin-status 语义呼应：两条在网主接口 up、
// spare 备用口物理 down、两条 admin-down 子接口 down）；bandwidth 单位 kbit/s
// （200GE=200000000）。经 SetStateDataXML 注入，不落配置树、get-config 不可见。
const DemoStateSeed = `<ifm xmlns="urn:huawei:params:xml:ns:yang:huawei-ifm">
    <interfaces>
      <interface>
        <name>200GE0/1/0</name>
        <dynamic>
          <oper-status>2</oper-status>
          <link-status>2</link-status>
          <physical-status>2</physical-status>
          <mac-address>00:e0:fc:12:34:01</mac-address>
          <bandwidth>200000000</bandwidth>
          <mtu>9216</mtu>
          <line-protocol-up-time>2026-07-01T08:30:00Z</line-protocol-up-time>
          <is-offline>false</is-offline>
          <sub-if-counts>1</sub-if-counts>
        </dynamic>
      </interface>
      <interface>
        <name>200GE0/1/1</name>
        <dynamic>
          <oper-status>2</oper-status>
          <link-status>2</link-status>
          <physical-status>2</physical-status>
          <mac-address>00:e0:fc:12:34:02</mac-address>
          <bandwidth>200000000</bandwidth>
          <mtu>9216</mtu>
          <line-protocol-up-time>2026-07-03T21:05:00Z</line-protocol-up-time>
          <is-offline>false</is-offline>
          <sub-if-counts>1</sub-if-counts>
        </dynamic>
      </interface>
      <interface>
        <name>200GE0/1/2</name>
        <dynamic>
          <oper-status>1</oper-status>
          <link-status>1</link-status>
          <physical-status>1</physical-status>
          <mac-address>00:e0:fc:12:34:03</mac-address>
          <bandwidth>200000000</bandwidth>
          <mtu>9216</mtu>
          <is-offline>true</is-offline>
          <sub-if-counts>0</sub-if-counts>
        </dynamic>
      </interface>
      <interface>
        <name>200GE0/1/0.1</name>
        <dynamic>
          <oper-status>1</oper-status>
          <link-status>1</link-status>
          <mac-address>00:e0:fc:12:34:01</mac-address>
          <bandwidth>200000000</bandwidth>
        </dynamic>
      </interface>
      <interface>
        <name>200GE0/1/1.1</name>
        <dynamic>
          <oper-status>1</oper-status>
          <link-status>1</link-status>
          <mac-address>00:e0:fc:12:34:02</mac-address>
          <bandwidth>200000000</bandwidth>
        </dynamic>
      </interface>
    </interfaces>
</ifm>`
