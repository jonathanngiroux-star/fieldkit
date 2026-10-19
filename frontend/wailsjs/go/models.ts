export namespace app {
	
	export class WizardField {
	    id: string;
	    label: string;
	    type: string;
	    required: boolean;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new WizardField(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.type = source["type"];
	        this.required = source["required"];
	        this.value = source["value"];
	    }
	}
	export class WizardStep {
	    id: string;
	    title: string;
	    body: string;
	    fields: WizardField[];
	    nextLabel: string;
	
	    static createFrom(source: any = {}) {
	        return new WizardStep(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.body = source["body"];
	        this.fields = this.convertValues(source["fields"], WizardField);
	        this.nextLabel = source["nextLabel"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace logic {
	
	export class FieldError {
	    question: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new FieldError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.question = source["question"];
	        this.message = source["message"];
	    }
	}
	export class Result {
	    visible_questions: string[];
	    errors: FieldError[];
	    next?: string;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.visible_questions = source["visible_questions"];
	        this.errors = this.convertValues(source["errors"], FieldError);
	        this.next = source["next"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

