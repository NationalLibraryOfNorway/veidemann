import { Injectable, inject } from '@angular/core';
import {MatSnackBar} from '@angular/material/snack-bar';


@Injectable({providedIn: 'root'})
export class SnackBarService {
  private snackBar = inject(MatSnackBar);


  public openSnackBar(message: string, action?: string, duration?: number) {
    this.snackBar.open(message, action, {
      duration: duration ? duration : 8000,
    });
  }
}
