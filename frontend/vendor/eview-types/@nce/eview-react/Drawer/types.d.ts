export type DrawerProps = {
    /**
     * 自定义id
     */
    id?: any;
    /**
     * 自定义类名
     */
    className?: string;
    /**
     * 抽屉的css样式
     */
    style?: React.CSSProperties;
    /**
     *  默认：`false`<br>
     * 是否显示抽屉
     */
    visible?: boolean;
    /**
     * 默认：`true`<br>
     * 是否显示遮罩
     */
    showMask?: boolean;
    /**
     * 默认：`true`<br>
     * 是否显示标题
     */
    showTitle?: boolean;
    /**
     * 默认：`true`<br>
     * 是否显示关闭按钮
     */
    showClose?: boolean;
    /**
     * 默认：`true`<br>
     * 是否点击遮罩关闭抽屉
     */
    isClickMask?: boolean;
    /**
     * 抽屉的标题
     */
    title?: string | Element;
    /**
     * 默认：`right`<br>
     *   抽屉的方向 默认right
     */
    placement?: 'top' | 'right' | 'bottom' | 'left';
    /**
     *  默认：`300px`<br>
     * 高度, 在 placement 为 top 或 bottom 时使用
     */
    height?: number;
    /**
     *  默认：`300px`<br>
     * 宽度，在 placement 为 left 或 right 时使用
     */
    width?: number;
    /**
     * 自定义关闭按钮和点击蒙层关闭的回调函数
     */
    onClose?: (isShowDrawer: boolean) => void;
    /**
     * 子节点
     */
    children: React.ReactNode;
    /**
     * 默认：`false`<br>
     * 关闭抽屉销毁内容区的子节点
     */
    destroyOnClose?: boolean;
    /**
     * 默认：`true`<br>
     *  指定 Drawers是否挂载在body节点, false 为挂载在当前 dom
     */
    isMountBody?: boolean;
    /**
     * 默认：`false`<br>
     * 指定抽屉大小可拖拽调整，横向抽屉调整宽度，纵向抽屉调整高度
     */
    sizeDraggable?: boolean;
    /**
     * 指定抽屉拖拽结束回调事件
     */
    onDragFinished?: (event: MouseEvent) => void;
    /**
     * 指定抽屉拖拽大小回调事件
     */
    onDragMove?: (size: number) => void;
    /**
     * 自定义抽屉内容的className，ev_drawer_content
     */
    contentClassName?: string;
    /**
    * 自定义动画持续时间
    */
    animationDuration?: number;
    /**
    * 自定义抽屉标题的tip内容
    */
    tipData?: string;
    /**
   * 自定义挂载节点，与isMountBody=false互斥
   */
    mountId?: string;
};
