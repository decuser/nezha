export namespace main {
	
	export class fileResponse {
	    path?: string;
	    content?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new fileResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.content = source["content"];
	        this.error = source["error"];
	    }
	}
	export class runResponse {
	    error?: string;
	    stopped?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new runResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.error = source["error"];
	        this.stopped = source["stopped"];
	    }
	}

}

