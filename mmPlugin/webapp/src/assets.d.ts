// Ассеты, попадающие в бандл через webpack (asset/inline отдаёт data-URI).
declare module '*.png' {
    const src: string;
    export default src;
}
