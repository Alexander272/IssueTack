const path = require('path');

module.exports = {
    mode: 'production',
    entry: './src/index.tsx',
    resolve: {
        modules: ['src', 'node_modules'],
        extensions: ['*', '.js', '.jsx', '.ts', '.tsx'],
    },
    module: {
        rules: [
            {
                test: /\.(js|jsx|ts|tsx)$/,
                exclude: /node_modules/,
                use: {
                    loader: 'babel-loader',
                    options: {
                        presets: [
                            '@babel/preset-typescript',
                            '@babel/preset-react',
                            [
                                '@babel/preset-env',
                                {
                                    modules: 'commonjs',
                                    targets: {browsers: ['last 2 versions', 'not dead']},
                                },
                            ],
                        ],
                    },
                },
            },
            {
                test: /\.css$/,
                use: ['style-loader', 'css-loader'],
            },
            // Иконка плагина для App Bar вшивается в бандл как data-URI: publicPath
            // равен '/', поэтому отдельный файл ассета отдавался бы из корня MM,
            // а не из каталога плагина. asset/inline отдаёт строку с base64 в
            // iconUrl — ровно то, что ждёт registerAppBarComponent.
            {
                test: /\.png$/,
                type: 'asset/inline',
            },
        ],
    },
    externals: {
        react: 'React',
        'react-dom': 'ReactDOM',
    },
    output: {
        path: path.join(__dirname, 'dist'),
        publicPath: '/',
        filename: 'main.js',
    },
};