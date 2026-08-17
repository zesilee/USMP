import React, { Component } from 'react';
import TreeNode from './TreeNode';
export type TreeProps = {
    /**
     * 组件唯一标识
     */
    id?: string;
    /**
     * 树节点数据，可指定多个根节点，数据格式：<br/>
     * text: 'Root'，id: '1000'，tip:'根节点'<br>
     * expanded：设置节点的展开状态，权重比expandedKeys高<br>
     * isLeaf:false<br>
     * children: [{text: 'NE_001', id: '1001'}]<br>
     * iconExpanded、iconCollapsed、iconLeaf、iconLeafArr、icon、showRightIcon、showRightIconArr、treeNodeSuffix、treeNodePrefix允许在data中单独为节点配置，用法与属性一致,权重比属性高<br>
     * hideRootCheckbox：节点的复选框是否隐藏<br>
     * draggable:节点是否可被拖拽<br>
     * show：节点是否展示<br>
     */
    data?: any[];
    /**
     * 节点标识属性名，默认为'id'，可以按业务需要修改
     */
    nodeKey?: string;
    /**
     * 默认：`false`<br>
     * 是否支持勾选(checkbox是否显示)
     */
    enableCheckbox?: boolean;
    /**
     * 默认：`check`<br>
     * 勾选框的类型： check - checkbox; radio - radio
     */
    selectBoxType?: string;
    /**
     * 默认：`level`<br>
     * radio选中方式
     */
    radioSelectMode?: string;
    /**
     * 默认：`true`<br>
     * select的同时是否同时触发checkbox或radio的选中事件
     */
    checkWhenSelect?: boolean;
    /**
     * 最外层div的内联样式
     */
    style?: React.CSSProperties;
    /**
     * 最外层div的class样式
     */
    className?: string;
    /**
     * 默认：`true`<br>
     * 是否支持多选
     */
    enableMultiSelect?: boolean;
    /**
     * 通过传入url的方式定义组件内部叶子节点图标,需配合iconExpanded和iconCollapsed一起使用
     * <br/>图片建议尺寸 16px*16px，支持url和Icon组件
     */
    iconLeaf?: string | React.ReactDOM;
    /**
     * 通过传入url的方式定义组件内部叶子节点展开时的图标,和iconLeaf互斥。需配合iconLeaf和iconCollapsed一起使用
     * <br/>图片建议尺寸 16px*16px，支持url和Icon组件
     */
    iconLeafArr?: any[];
    /**
     * 通过传入url的方式定义组件非叶子节点展开时的图标,需配合iconLeaf和iconCollapsed一起使用
     * <br/>图片建议尺寸 16px*16px，支持url和Icon组件
     */
    iconExpanded?: string | React.ReactDOM;
    /**
     * 通过传入url的方式定义组件非叶子节点关闭时的图标,需配合iconLeaf和iconExpanded一起使用
     * <br/>图片建议尺寸 16px*16px，支持url和Icon组件
     */
    iconCollapsed?: string | React.ReactDOM;
    /**
     * 设置叶子节点图标类名
     */
    iconLeafClassName?: string;
    /**
     * 设置展开节点图标类名
     */
    iconExpandedClassName?: string;
    /**
     * 设置收起节点图标类名
     */
    iconCollapsedClassName?: string;
    /**
     * data中写入，可以单独为节点设置样式
     */
    treeNodeStyle?: any;
    /**
     * 未知属性-暂未开放
     */
    onVerticalScroll?: any;
    /**
     * 用于子组件expandedKeys改变回传
     */
    onRef?: Function;
    /**
     * 设置默认选中的节点，其值为节点标识，如：['1000', '1001']
     */
    selectedKeys?: any[];
    /**
     * 设置默认勾选的节点，需要先设置enableCheckbox=true，其值为节点标识，如：['1000', '1001']
     */
    checkedKeys?: any[];
    /**
     * 默认选中radio的key值
     */
    radioKey?: any;
    /**
     * 设置默认展开的节点，其值为节点标识，如：['1000', '1001']
     */
    expandedKeys?: any[];
    /**
     * 默认：`false`<br>
     * 设置展开全部节点
     */
    expandAll?: boolean;
    /**
     * 默认：`false`<br>
     * 设置关闭全部节点
     */
    cancelAll?: boolean;
    /**
     * 默认：`false`<br>
     * 设置组件的灰化
     */
    disabled?: boolean;
    /**
     * 节点单击选中时事件回调<br>
     * 签名:`function(selectedKeys: array, node: object, event:object) => void`<br>
     *  {array} selectedKeys 储存有点击节点eventKey的数组
     *  {object} node 被点击节点的节点信息
     */
    onSelect?: (selectedKeys: any[], node: any, event: React.MouseEvent<HTMLAnchorElement>) => void;
    /**
     * 节点勾选或去选时事件回调，需要先设置enableCheckbox=true<br>
     * 签名：`function(checkedKeys: array, node: object) => void`<br>
     * {array} checkedKeys 储存有已勾选节点的eventKey的数组<br>
     *  {object} node 被勾选节点的节点信息
     */
    onCheck?: (checkedKeys: any[], node: any, checkedNodeArr?: Array<TreeNode>) => void;
    /**
     * 节点展开或折叠时事件回调，只有非叶子节点才会触发<br>
     * 签名：`function(expandedKeys: array, node: object) => void`<br>
     * {array} expandedKeys 储存有展开节点eventKey的数组<br>
     * {object} node 被展开节点的节点信息
     */
    onExpand?: (checkedKeys: any[], node: any) => void;
    /**
     * Event callback for double click on the node<br>
     * 签名：`function(nodeKey: any, node: object) => void`<br>
     *  {any} nodeKey The eventKey of the node that was double-clicked<br>
     *  {object} node the node object information
     */
    onNodeDoubleClick?: (eventKey: string | number, node: any, event: React.MouseEvent<HTMLAnchorElement>) => void;
    /**
     * Event callback for right click on the node<br>
     * 签名：`function(nodeKey: any, node: object, event:object) => void`<br>
     *  {any} nodeKey The eventKey of the node that was right-clicked<br>
     *  {object} node the node object information
     */
    onNodeRightClick?: (node: any, event: React.MouseEvent<HTMLLIElement>) => void;
    /**
     * Event callback for right icon click on the node<br>
     * 签名：`function( node: object , event:object) => void`<br>
     * node: The node information of the clicked node
     */
    onClickRightIcon?: (node: any, event: React.MouseEvent<HTMLAnchorElement>) => void;
    /**
     * 设置需要聚焦的节点，外层容器高度不足会滚动目标节点至父元素中央视口<br>
     * focusNode={{ key: this.state.selectedKeys[0] }}
     */
    focusNode?: object;
    /**
     * 默认：`false`<br>
     * 是否启用/禁用树的水平滚动。
     */
    enableScroll?: boolean;
    /**
     * 节点文本右侧图标,支持url和Icon组件
     */
    showRightIcon?: string | React.ReactDOM;
    /**
     * 默认：`false`<br>
     * 设定为true时，未展开的子项dom不会渲染到页面上
     */
    lazyLoad?: boolean;
    /**
     * 默认：`false`<br>
     * 设置节点可拖拽（IE>8)
     */
    draggable?: boolean;
    /**
     * 应用于拖拽元素，当拖拽开始时调用
    */
    onDragStart?: (event: React.DragEvent<HTMLAnchorElement>, node?: any, dragKey?: number) => void;
    /**
     *  应用于拖拽元素，当拖拽结束时调用
     */
    onDragEnd?: (event: React.DragEvent<HTMLAnchorElement>, node?: any, dragKey?: number) => void;
    /**
     * 应用于拖拽元素，当鼠标离开拖拽元素是调用
     */
    onDragLeave?: (event: React.DragEvent<HTMLAnchorElement>, node?: any, dragKey?: number) => void;
    /**
     * 应用于目标元素，当拖拽元素进入时调用
     */
    onDragEnter?: (event: React.DragEvent<HTMLAnchorElement>, node?: any, dragKey?: number) => void;
    /**
     * 应用于目标元素，当停留在目标元素上时调用
     */
    onDragOver?: (event: React.DragEvent<HTMLAnchorElement>, node?: any, dragKey?: number) => void;
    /**
     * 应用于目标元素，当在目标元素上松开鼠标时调用
    */
    onDrop?: (event: React.DragEvent<HTMLAnchorElement>, node?: any, dragType?: string) => void;
    /**
     * 默认：`false`<br>
     * 如果为超多级树节点，且外层容器不足以容纳全部的树节点时，可添加此属性
     */
    superLevel?: boolean;
    /**
     * 默认：`true`<br>
     * 子节点置灰，勾选状态是否还与父节点联动。若为false,父节点选中则置灰的子节点不会被勾选
     */
    disabledLinkage?: boolean;
    /**
     * 默认：`true`<br>
     * 父子节点的联动是否保持，默认为 true,当设置为 false,父子节点之间选中状态互不影响<br>
     * （注意：此属性名有误后续调整）
     */
    disabelCheckAssociated?: boolean;
    /**
     * 默认：`false`<br>
     * selectBoxType(勾选框的类型)为radio时，点击选中节点是否取消选中状态
     */
    radioCancelable?: boolean;
    /**
     * 默认：`default`<br>
     * 设置节点右侧自定义后缀的展示状态,默认default：直接显示且紧跟在节点后面。hover：划入显示且右对齐排列
     */
    nodeSuffixTrigger?: 'default' | 'hover';
    /**
     * 默认：`false`<br>
     * 设置节点右侧自定义后缀的展示状态为hover时，节点选中，是否长显后缀
     */
    selectedAlwaysShow?: boolean;
    /**
     * 自定义节点右侧内容后缀。全局节点统一配置，也可以在data中给各节点单独配置。
     */
    treeNodeSuffix?: React.ReactElement;
    /**
     * 自定义节点左侧内容前缀。全局节点统一配置，也可以在data中给各节点单独配置。
     */
    treeNodePrefix?: React.ReactElement;
    /**
     * 节点文本右侧图标,允许设置多个，支持url和Icon组件。优先级大于showRightIcon,也可以在data中给各节点单独配置。
     */
    showRightIconArr?: string[] | React.ReactElement[];
    /**
     * 叶子节点多个图标场景下，设置各自类名；类名与图标一一对应。也可以在data中给各节点单独配置。
     */
    iconLeafArrClassName?: string[];
    /**
     * 节点多个右侧图标场景下，设置各自类名；类名与图标一一对应。也可以在data中给各节点单独配置。
     */
    showRightIconArrClassName?: string[];
    /**
     * 默认：`true`<br>
     * 节点点击选中会触发其复选框选中(节点的onSelect事件中默认触发onCheck)
     */
    selectTriggerCheck?: boolean;
    /**
     * 为节点文本设置样式。也可以在data中给各节点单独配置
     */
    treeTextStyle?: React.CSSProperties;
    /**
     * 异步加载数据(此回调在onExpand且节点为收起态时触发), 参数itemData为当前节点数据; 参数callback,传入需要添加的数据
     */
    loadData?: (itemData: any, callback: any) => void;
    /**
     * 默认：`false`<br>
     * 是否展示节点之间的连线
     */
    connectLine?: boolean;
    /**
     * 默认：`true`<br>
     * 是否允许同级节点多个展开，权重小于expandAll属性;
     */
    enableMultiExpand?: boolean;
    /**
     * 传入视口高度，虚拟滚动。仅支持传入number类型的像素高度
     */
    height?: number | string;
    /**
     * 默认：`true`<br>
     * 当使用hideRootCheckbox隐藏掉复选框后，节点是否还支持select选中
     */
    enableSelectWithHideRootCheckbox?: boolean;
};
export interface TreeState {
    keyIndexMap: any;
    anchorKey: any;
    checkedKeys: any[];
    selectedKeys: any[];
    expandedKeys: any[];
    halfCheckedKeys: any[];
    radioKey: any;
    isKeyBoardSelect: boolean;
    data: any;
    visibleData: Array<string | number>;
    visibleStart: number;
    flatData: Array<string | number>;
}
export default class Tree extends Component<TreeProps, TreeState> {
    addToSelected: boolean;
    preScrollTop: number;
    parentTreeNode: any;
    endOfTree: any;
    orientation: boolean;
    allNodeArr: Array<any>;
    static defaultProps: {
        nodeKey: string;
        enableCheckbox: boolean;
        selectBoxType: string;
        radioSelectMode: string;
        checkWhenSelect: boolean;
        enableMultiSelect: boolean;
        expandAll: boolean;
        disabled: boolean;
        cancelAll: boolean;
        enableScroll: boolean;
        lazyLoad: boolean;
        draggable: boolean;
        checkedKeys: any[];
        expandedKeys: any[];
        selectedKeys: any[];
        superLevel: boolean;
        disabledLinkage: boolean;
        disabelCheckAssociated: boolean;
        radioCancelable: boolean;
        nodeSuffixTrigger: string;
        selectedAlwaysShow: boolean;
        selectTriggerCheck: boolean;
        connectLine: boolean;
        enableMultiExpand: boolean;
        enableSelectWithHideRootCheckbox: boolean;
    };
    constructor(props: any);
    /**
     * 刷新state中依赖data变更的变量状态。
     * 1、constructor会初始化state，执行此方法。
     * 2、loadData属性在TreeNode节点expand的时候，会变更data，执行此方法。
     * @param newData 变更后的data
     * @param isMount 是否为constructor时调用，如果是，部分变量的初始值从props读取，否则从state中读取
     * @returns isMount:true会有返回值，定时初始化的state变量
     */
    updateStateByDataChange: (newData: any, isMount?: boolean) => any;
    /**
     *
     * @param filterArray 查找的得到结果Array
     * @param result 用戶輸入需要搜索的内容
     * @param data  总数据
     */
    find: (filterArray: any, result: any, data: any) => void;
    /**
     *
     * @param result 用戶輸入需要搜索的内容
     * @param callback 可选参数，回调函数
     * @returns 返回具有搜索内容的节点
     */
    findNodes(result: any): any[];
    /**
     * 通过子节点查找其全部父节点
     * @param id 子节点id
     * @param list data
     * @param result 全部节点的Array
     * @returns
     */
    findP(node: any, list?: any[], result?: any[]): boolean | any[];
    /**
     *
     * @param filterArray 查找到结果的array
     * @param result  用户输入的搜索内容
     * @param data  数据
     */
    findLevel: (filterArray: any, result: any, data: any) => void;
    /**
     *
     * @param strText  处理之前的文本
     * @param result   查询的词条
     * @returns  返回处理后的文本dom
     */
    handleText(strText: any, result: any): any;
    /**
     * 查找高亮符合筛选条件的字段
     * @param data 总数据
     * @param result 用戶輸入需要搜索的内容
     */
    loop: (data: any, result: any) => any;
    /**
     * 返回具有搜索内容的节点以及它们的父级节点
     * @param result 用户输入的搜索内容
     * @returns 返回值数组，包含符合条件的节点以及父级节点
     */
    findLevelNodes(result: any): any[];
    /**
     * @description 返回树数据所有的节点nodeKey（节点标识属性名）
     * @param {Array} treeData 树数据
     * @param {string | number} nodeKey 节点nodeKey（节点标识属性名）
     * @returns {Array}
     */
    static getAllTreeNodeId(treeData: any, nodeKey: any): any[];
    /**
     * @version 3.8.21
     * @description 将选中节点的信息进行简化，返回（例如：父节点和子节点全部选中，只返回父节点即可）。
     * @param {array} checkedKeys 传入选中节点
     * @returns {object} 返回对象，包含了简化后的checkedKeys和平铺一维data
     */
    getSimpleCheckedNodeArr: (checkedKeys?: any[]) => {
        checkedKeys: any[];
        data: any[];
    };
    /**
     * @version 3.8.21
     * @description 只返回选中节点中的叶子节点。
     * @param {array} checkedKeys 传入选中节点
     * @returns {object} 返回对象，包含了checkedKeys和平铺一维data
     */
    getLeafCheckedNodeArr: (checkedKeys?: any[]) => {
        checkedKeys: any[];
        data: any[];
    };
    containRef: React.RefObject<HTMLDivElement>;
    containerHeight: number;
    resizeObserver: ResizeObserver | null;
    iteratorChildren: (data: any, itemAnalysisFun: any, isIteratorChildren: any) => any;
    fillKeyIndexMapAndInitNodeState: (keyIndexMap: any, data: any, prefixIndexArray: any) => void;
    getInitExpandKeys(data: any, expandedKeys: any, expandAll: any, cancelAll: any): any;
    getInitialExpandedKeys(value: any, expandedKeys: any, path?: any[]): any;
    /**
     * 初始化或者data发生变化时，不允许同级节点多个展开的处理函数
     * @param value
     * @param expandedKeys
     * @returns
     */
    handleSingleExpand: (value: any, expandedKeys: any) => void;
    /**
     * 当节点手动点击展开时，不允许同级节点多个展开的处理函数
     * @param value
     * @param expandedKeys
     * @param node
     */
    handleSingleExpandWhenExpand: (value: any, expandedKeys: any, node: any) => void;
    initExpandedkeys(value: any, expandedKeys: any, parentKey: any, path: any): void;
    iterator(value: any, checkedKeys: any, parentKey: any, path: any): void;
    getInitialCheckedKeys(value: any, checkedKeys: any, path?: any[]): any;
    deleteCheckedkeysNotInData(treeNodeIds: any, checkedKeys: any): any[];
    iteratorData(data: any, parentKey: any, value: any, checkedArr: any, path: any, hideRootCheckbox?: boolean): void;
    getHalfCheckedKeys(value: any, checkedArr: any, path?: any[]): any[];
    /** *****展开所有节点*******/
    getAllExpandKeys(value: any, keys?: any[], parentKey?: string, path?: any[]): any[];
    iteratorExpand(value: any, expandedKeys: any, parentKey: any, path: any): void;
    expandAll(): void;
    cancelAll(): void;
    /**
     * Get the list of expanded nodes keys
     * @returns array of expanded nodes keys
     */
    getExpandedKeys(): any[];
    /**
     * Get the list of selected nodes keys
     * @returns array of selected nodes keys
     */
    getSelectedKeys(): any[];
    /**
     * Get the list of checked nodes keys
     * @returns array of checked nodes keys
     */
    getCheckedKeys(): any;
    /**
     * Get the list of expanded, selected and checked nodes keys
     * @returns array of expanded, selected and checked nodes keys
     */
    getValue(): {
        expandedKeys: any[];
        selectedKeys: any[];
        checkedKeys: any[];
    };
    getNextPropsExpandKeys(data: any, expandAll: any, cancelAll: any, expandedKeys: any): any;
    handleSelect: (node: any, event: any) => void;
    handleCheckWhenSelect: (node: any) => void;
    handleShiftSelectKeys: (node: any, event: any) => void;
    getMultiSelectedKeys: (selectedKey: any, anchorKey: any) => any[];
    handleRightIconClick: (node: any, event: React.MouseEvent<HTMLAnchorElement>) => void;
    handleNodeDoubleClick: (node: any, event: React.MouseEvent<HTMLAnchorElement>) => void;
    handleNodeRightClick: (node: any, event: React.MouseEvent<HTMLLIElement>) => void;
    /**
     * 无复选框的节点id应当从checkedKeys中过滤出去
     * @param param0
     * @returns
     */
    filterNodeByShowCheckBox: ({ checkedKeys, halfCheckedKeys }: {
        checkedKeys: any;
        halfCheckedKeys: any;
    }) => {
        checkedKeys: any[];
        halfCheckedKeys: any;
    };
    handleCheck: (node: any) => void;
    getAllCheckedState: (eventKey: any) => {
        checkedKeys: any[];
        halfCheckedKeys: any[];
    };
    getNodeCheckStateByChildren: (parentNode: any, checkedKeys: any, halfCheckedKeys: any, isAdd: any) => void;
    deleteKeysInArray: (node: any, checkedKeys: any, halfCheckedKeys: any) => void;
    isOneChildrenChecked: (node: any, checkedKeys: any, halfCheckedKeys: any) => any;
    isNotAllChildrenChecked: (node: any, checkedKeys: any) => any;
    deleteOrAddChildrenKeyToArray: (node: any, array: any, isAdd: any) => void;
    addOrDeleteItemToArrayIfNotInArray: (item: any, array: any, isAdd: any) => void;
    getNodeArrayByKey: (key: any) => any[];
    onRadioChange: (node: any) => void;
    unCheckOthers(checkedKeys: any, node: any): void;
    handleExpand: (node: any) => void;
    onDragStart: (info: any, node: any) => void;
    onDragEnd: (info: any, node: any) => void;
    onDragEnter: (info: any, node: any) => void;
    onDragLeave: (info: any, node: any) => void;
    onDragOver: (info: any, node: any) => void;
    onDrop: (info: any, node: any, dragType: any) => void;
    createChildNodes(data: any, path?: any[], level?: number): any;
    upDownKeyDown: (event: any) => void;
    getParent: (dom: any, selector: any, endSelector: any) => any;
    isFocuseTree: () => boolean;
    getLastNodeId: () => any;
    getNextNode: () => any;
    fillChildrenNode: (container: any, children: any) => void;
    componentDidMount(): void;
    componentWillUpdate(nextProps: any, nextState: any, nextContext: any): void;
    componentWillUnmount(): void;
    componentWillReceiveProps(nextProps: Readonly<TreeProps>, nextContext: any): void;
    componentDidUpdate(prevProps: any, prevState: any): void;
    handleKeyDown: (e: any) => void;
    /**
     * 平铺非收起且show=true的树结构数据
     * @param data
     * @returns
     */
    flatVisibleTree: (data: any, expandedKeys: any, isMount?: boolean) => any;
    getVisibleData: (scrolledNodeNum?: any, newFlatData?: (string | number)[], isMount?: boolean) => any;
    initResizeObserver: () => void;
    destroyResizeObserver: () => void;
    getContainerHeight: () => number;
    render(): React.JSX.Element;
    handleScroll: (e: any) => void;
    virtualScrollFunc: (e: any) => void;
}
