import CopyWebpackPlugin from 'copy-webpack-plugin';
import ForkTsCheckerWebpackPlugin from 'fork-ts-checker-webpack-plugin';
import path from 'path';
import fs from 'fs';
import { fileURLToPath } from 'url';
import ReplaceInFileWebpackPlugin from 'replace-in-file-webpack-plugin';
import { Configuration, ExternalItemFunctionData } from 'webpack';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const SOURCE_DIR = path.join(__dirname, 'src');
const DIST_DIR = path.join(__dirname, 'dist');
const packageVersion = JSON.parse(fs.readFileSync(path.join(__dirname, 'package.json'), 'utf-8')).version;

const config = async (env: Record<string, unknown>): Promise<Configuration> => {
  const isProduction = Boolean(env.production);

  return {
    target: 'web',
    mode: isProduction ? 'production' : 'development',
    devtool: isProduction ? 'source-map' : 'eval-source-map',
    context: __dirname,

    entry: {
      module: path.join(SOURCE_DIR, 'module.ts'),
    },

    output: {
      filename: '[name].js',
      path: DIST_DIR,
      libraryTarget: 'amd',
      publicPath: '/',
    },

    externals: [
      'lodash',
      'jquery',
      'moment',
      'slate',
      'emotion',
      '@emotion/react',
      '@emotion/css',
      'prismjs',
      'slate-plain-serializer',
      '@grafana/slate-react',
      'react',
      'react-dom',
      'react-redux',
      'redux',
      'rxjs',
      'react-router-dom',
      'd3',
      'angular',
      '@grafana/ui',
      '@grafana/runtime',
      '@grafana/data',
      (data: ExternalItemFunctionData, callback: (err?: Error | null, result?: string) => void) => {
        const prefix = 'grafana/';
        if (data.request.indexOf(prefix) === 0) {
          return callback(undefined, data.request.replace(prefix, ''));
        }
        callback();
      },
    ],

    plugins: [
      new CopyWebpackPlugin({
        patterns: [
          { from: 'img', to: 'img', noErrorOnMissing: true },
          { from: 'plugin.json', to: '.' },
          { from: 'CHANGELOG.md', to: '.', noErrorOnMissing: true },
          { from: 'README.md', to: '.' },
        ],
      }),
      new ForkTsCheckerWebpackPlugin({
        async: isProduction ? false : true,
        typescript: {
          configFile: path.join(__dirname, 'tsconfig.json'),
        },
      }),
      new ReplaceInFileWebpackPlugin([
        {
          dir: DIST_DIR,
          files: ['plugin.json', 'README.md'],
          rules: [
            {
              search: '%VERSION%',
              replace: packageVersion,
            },
            {
              search: '%TODAY%',
              replace: new Date().toISOString().substring(0, 10),
            },
          ],
        },
      ]),
    ],

    resolve: {
      extensions: ['.js', '.jsx', '.ts', '.tsx'],
    },

    module: {
      rules: [
        {
          test: /\.[tj]sx?$/,
          exclude: /node_modules/,
          use: {
            loader: 'swc-loader',
            options: {
              jsc: {
                baseUrl: SOURCE_DIR,
                target: 'es2018',
                loose: false,
                parser: {
                  syntax: 'typescript',
                  tsx: true,
                  decorators: false,
                  dynamicImport: true,
                },
              },
            },
          },
        },
        {
          test: /\.css$/,
          use: ['style-loader', 'css-loader'],
        },
        {
          test: /\.(scss|sass)$/,
          use: ['style-loader', 'css-loader', 'sass-loader'],
        },
        {
          test: /\.(png|jpe?g|gif|svg)$/,
          type: 'asset/resource',
          generator: {
            filename: 'img/[name].[hash:8][ext]',
          },
        },
        {
          test: /\.(woff|woff2|eot|ttf|otf)$/,
          type: 'asset/resource',
          generator: {
            filename: 'fonts/[name].[hash:8][ext]',
          },
        },
      ],
    },
  };
};

export default config;
