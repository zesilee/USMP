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
