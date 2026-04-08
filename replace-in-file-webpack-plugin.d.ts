declare module 'replace-in-file-webpack-plugin' {
  interface ReplaceConfig {
    search: string | RegExp;
    replace: string | ((match: string) => string);
  }

  interface ReplaceInFileOptions {
    dir?: string;
    files?: string[];
    test?: RegExp | RegExp[];
    rules: ReplaceConfig[];
  }

  class ReplaceInFileWebpackPlugin {
    constructor(options: ReplaceInFileOptions[]);
    apply(compiler: any): void;
  }

  export = ReplaceInFileWebpackPlugin;
}
