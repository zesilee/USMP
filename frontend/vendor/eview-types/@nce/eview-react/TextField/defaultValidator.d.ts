declare const defaultValidator: {
    min: (num: number) => (value: string) => {
        result: boolean;
        message: import("react").JSX.Element;
    };
    max: (num: number) => (value: string) => {
        result: boolean;
        message: import("react").JSX.Element;
    };
    integer: () => (value: string) => {
        result: boolean;
        message: import("react").JSX.Element;
    };
    range: (min: number, max: number) => (value: string) => {
        result: boolean;
        message: import("react").JSX.Element;
    };
    rangeAndInteger: (min: number, max: number) => (value: string) => {
        result: boolean;
        message: import("react").JSX.Element;
    };
    number: () => (value: string) => {
        result: boolean;
        message: import("react").JSX.Element;
    };
    email: () => (value: string) => {
        result: boolean;
        message: import("react").JSX.Element;
    };
    digit: () => (value: string) => {
        result: boolean;
        message: import("react").JSX.Element;
    };
    url: () => (value: string) => {
        result: boolean;
        message: import("react").JSX.Element;
    };
    alpha: () => (value: string) => {
        result: boolean;
        message: import("react").JSX.Element;
    };
    regex: (reg: string) => (value: string) => {
        result: boolean;
        message: import("react").JSX.Element;
    };
    postfix: (str: string) => (value: string) => {
        result: boolean;
        message: import("react").JSX.Element;
    };
    ipv4: () => (value: string) => {
        result: boolean;
        message: import("react").JSX.Element;
    };
    ipv6: () => (value: string) => {
        result: boolean;
        message: import("react").JSX.Element;
    };
    creditCard: () => (value: string) => {
        result: boolean;
        message: import("react").JSX.Element;
    };
    equalTo: (str: string) => (value: string) => {
        result: boolean;
        message: import("react").JSX.Element;
    };
    notEqualTo: (str: string) => (value: string) => {
        result: boolean;
        message: import("react").JSX.Element;
    };
    minLength: (length: number) => (value: string) => {
        result: boolean;
        message: import("react").JSX.Element;
    };
    maxLength: (length: number) => (value: string) => {
        result: boolean;
        message: import("react").JSX.Element;
    };
    rangeLength: (minLength: number, maxLength: number) => (value: string) => {
        result: boolean;
        message: import("react").JSX.Element;
    };
};
export default defaultValidator;
