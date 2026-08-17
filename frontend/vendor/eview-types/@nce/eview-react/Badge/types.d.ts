export type BadgeProps = {
    /**
     * 自定义id名
     */
    id?: string;
    /**
     *通过添加类的方式控制整个Badge的样式（控制最外层div）
     */
    className?: string;
    /**
     * 通过添加style的方式控制整个Badge的样式（控制最外层的div）
     */
    style?: React.CSSProperties;
    /**
     * 徽标的自定义类名
     */
    badgeClassName?: string;
    /**
     * 徽标的自定义样式
     */
    badgeStyle?: React.CSSProperties;
    /**
     * 不展示数字，只有一个小红点
     * @defaultValue false
     */
    dot?: boolean;
    /**
     * 最大值，超过最大值会显示{max}+,仅当content为数字时有效
     * @defaultValue 99
     */
    max?: number;
    /**
     * 徽标的内容，,如果null| undefined |'' 或不传，则不显示徽标
     */
    content?: React.ReactNode | string | number;
    /**
     * 设置状态点的位置偏移
     */
    offset?: [number, number];
    /**
     * 当数值为 0 时，是否展示 Badge
     */
    showZero?: boolean;
    /**
     * 设置 Badge 为状态点
     *
     */
    status?: 'default' | 'success' | 'error' | 'warning' | 'off';
    /**
     * 在设置了 status 的前提下有效，设置状态点的文本
     */
    text?: React.ReactNode | string;
    /**
     * React的children
     */
    children?: React.ReactNode;
};
