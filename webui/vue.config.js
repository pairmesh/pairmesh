// https://cli.vuejs.org/zh/config/#pages
module.exports = {
    assetsDir: 'assets',
    // CDN address
    publicPath: '/',
    pages: {
        main: {
            entry: 'src/main.js',
            filename: 'index.html'
        },
        console: {
            entry: 'src/console/main.js',
            filename: 'console/index.html'
        },
        login: {
            entry: 'src/login/main.js',
            filename: 'login/index.html'
        }
    },
    devServer: {
        proxy: {
            '/api/': {
                target: 'http://localhost:2823',
                changeOrigin: true
            },
        }
    }
}