var dns = require('dns');

exports.handler = function(data) {
    return new Promise((resolve, reject) => {
        try {
            dns.lookup(data.toString('utf8'), (err, address, family) => {
                if (!!err) {
                    reject(err);
                    return
                }

                resolve(address);
            });        
        } catch(e) {
            reject(e);
        }
    });
}
