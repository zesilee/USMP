export default class TableCalc {
    static COLUMN_MIN_WIDTH: number;
    static isRowDataAnObject: (data: any) => boolean;
    /**
     * 转换数据结构，如果数据时一个对象，
     * @param columns
     * @param data
     * @returns
     */
    static convertDataStructure: (columns: any, data: any) => any;
    static typeName(obj: any): any;
    /**
     * 校验数据是否合法，如果是通过key来匹配数据的，数据为空，不抛出异常
     * @param columns
     * @param rows
     * @returns
     */
    static verifyData(columns: any, rows: any): boolean;
    static initColumns(columns: any, rowExpand: any): void;
    static initData(columns: any, rows: any, keyIndex: any, rowExpand: any, groupHeaders: any, stateData?: any): any;
    static initRow(columns: any, row: any, rowIndex: any, keyIndex: any, stateData?: any): any;
    static guessGetter(obj: any): (value: any) => any;
    static columnIdsFromTitle(columns: any, columnNames: any): any[];
    static isFixSize(columns: any): boolean;
    static getColumnOffset(obj: any, left: any): {
        movePx: number;
        lastCid: number;
    };
    /**
     * 列宽最大最小值判断
     * @param column
     */
    static columnWidthFix(column: any): void;
    static getDragColumnWidth(column: any, obj: any, minWidth: any): any;
    static getDragColumnWidthFreezed(column: any, obj: any, minWidth: any): any;
}
