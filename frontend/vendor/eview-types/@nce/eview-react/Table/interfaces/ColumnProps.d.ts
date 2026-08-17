import React from "react";
import type { HelpTipProps } from '../../HelpTip';
export type AlignType = 'center' | 'left' | 'right';
export type SortOrder = 'asc' | 'desc' | 'origin';
export type RenderType = 'check_box' | 'check_box_group' | 'date_picker' | 'time_selector' | 'progress_bar' | 'radio_group' | 'select' | 'input_select' | 'text_field' | 'custom';
export default interface ColumnProps {
    key?: string;
    id?: string | number;
    cid?: string | number;
    title?: number | string | React.ReactNode;
    titleTipShow?: string;
    titleTipData?: object;
    titleClassName?: string;
    width?: string | number;
    align?: AlignType;
    allowSort?: boolean;
    display?: boolean;
    /**
     * displayPolicy 为'never'时，列将永久隐藏
     */
    displayPolicy?: string;
    renderType?: RenderType;
    render?: (cellValue: any, rowData: Array<any>, options: any, row: any, idEdit: any) => any;
    getCompareValue?: (v: any) => any;
    options?: any;
    tipFormatter?: Function | string;
    freezeCol?: boolean;
    isMovable?: boolean;
    disableOrderChange?: boolean;
    isEditable?: boolean;
    validator?: (v: any) => boolean;
    sort?: SortOrder;
    filter?: object;
    embeddedFilter?: object;
    help?: HelpTipProps;
    titleComponentToText?: string;
    minWidth?: number;
    maxWidth?: number;
}
