import weather = require("weather-js");

export function handler(data: any) {
    return new Promise<any>((resolve, reject) => {
        weather.find({search: data, degreeType: 'C'}, function(err: any, result: any) {
            if(err) reject(err);
           
            resolve(result);
          });
    });
}