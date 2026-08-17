import React, { Component } from 'react';
interface ICrumb {
    url?: string;
    title: string;
    enable?: boolean;
    icon?: string | React.ReactElement;
    style?: React.CSSProperties;
}
export type CrumbsProps = {
    /**
     * 自定义id
     * 例如：
     * id='crumbsId'
     */
    id?: string;
    /**
     * 自定义className
     * 例如：
     * className='crumbsClassName'
     */
    className?: string;
    /**
     *自定义面包屑内联样式
     */
    style?: React.CSSProperties;
    /**
     * 自定义面包屑标题
     * 例如：
     * title='当前位置'
     */
    title?: string;
    /**
     * 默认：`>`<br>
     * 自定义分隔符
     */
    seprator?: string;
    /**
     * 自定义分割图标，
     * 当需要自定义面包屑分割图标时，需传入分割图标的url，
     * 不传入则采用默认分割图片。
     */
    splitIcon?: string;
    /**
     * 面包屑配置数据
     * <br>const ICrumbData = [{
     * <br>title: string;文本
     * <br>enable?: boolean;是否禁用
     * <br>icon?: string;设置图标
     * <br>url?: string;路由跳转
     * }];
     */
    data: ICrumb[];
    /**
     * 定义带有链接的面包屑点击事件的回调方法<br>
     * data  触发当前点击目标元素的用户传入数据<br>
     * event dom原生事件.
     */
    onClick?: (data: ICrumb, event: React.MouseEvent | React.KeyboardEvent) => void;
    /**
     * 每项 style
     */
    itemStyle?: React.CSSProperties;
    /**
     * 默认：`6`<br>
     *多级节点, 面包屑数据data的length超出此值展示下拉菜单
     **/
    countLimit?: number;
    /**
     * 默认：`false`<br>
     * 设置是否显示面包屑鼠标移入显示的tips
     */
    itemTip?: boolean;
};
export default class Crumbs extends Component<CrumbsProps> {
    id: string;
    static defaultProps: {
        seprator: string;
        countLimit: number;
        itemTip: boolean;
    };
    getTitle(): React.JSX.Element;
    handleKeyPress: (child: ICrumb, e: React.KeyboardEvent) => void;
    handleClick: (e: React.KeyboardEvent, child: ICrumb) => void;
    onClick: (child: ICrumb) => (event: React.MouseEvent) => void;
    getCrumbsList: () => React.JSX.Element[];
    getMultiCrumbsList: () => React.JSX.Element[];
    moreDataItemClick: (data: any, e: any) => void;
    render(): React.JSX.Element;
}
export {};
