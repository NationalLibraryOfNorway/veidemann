import {Injectable} from '@angular/core';
import {Resolve} from '@angular/router';
import {Observable} from 'rxjs';
import {Level, levels} from '../../../shared/models';

@Injectable({
  providedIn: 'root'
})
export class LogResolver implements Resolve<Level[]> {

  resolve(): Observable<Level[]> | Promise<Level[]> | Level[] {
    return levels;
  }
}
