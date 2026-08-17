export default class TableRenderer {
    /**
     * 渲染文本
     *
     * @static
     * @param {String} data 文本
     * @returns {String}
     *
     * @memberof TableRenderer
     */
    static renderText(data: any): any;
    static renderUndefined(): string;
    /**
     * 渲染数字
     *
     * @static
     * @param {Number} data 数字
     * @returns {String}
     *
     * @memberof TableRenderer
     */
    static renderNumber(data: any): any;
    /**
     * 渲染日期
     *
     * @static
     * @param {Date} data 日期
     * @returns {String}
     *
     * @memberof TableRenderer
     */
    static renderDateTime(data: any): string;
    /**
     * 渲染到原始字符串
     *
     * @static
     * @param {Object} data 原始数据
     * @returns {String}
     */
    static renderRaw(data: any): string;
    static renderArray(data: any): any;
}
