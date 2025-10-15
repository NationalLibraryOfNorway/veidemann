import {Component, Input} from '@angular/core';
import {ConfigObject} from '../../../../../shared/models';
import {AuthService, SnackBarService} from '../../../../../core';
import {UrlFormatPipe} from '../../../../../shared/pipes/url-format.pipe';
import {DatePipe} from '@angular/common';
import {MatIcon} from '@angular/material/icon';
import {MatButtonModule} from '@angular/material/button';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatChipsModule} from '@angular/material/chips';
import {MatTooltip} from '@angular/material/tooltip';

@Component({
  selector: 'app-seed-meta-preview',
  templateUrl: './seed-meta-preview.component.html',
  styleUrls: ['./seed-meta-preview.component.css'],
  imports: [
    DatePipe,
    MatButtonModule,
    MatChipsModule,
    MatFormFieldModule,
    MatIcon,
    MatTooltip,
    UrlFormatPipe
  ],
  standalone: true
})
export class SeedMetaPreviewComponent {

  @Input()
  configObject: ConfigObject;

  constructor(private snackBarService: SnackBarService,
              private authService: AuthService) {
  }

  get canShowAnnotation() {
    return this.authService.isAdmin() || this.authService.isOperator() || this.authService.isCurator();
  }

  copyIdToClipboard() {
    const dummy = document.createElement('textarea');
    document.body.appendChild(dummy);
    dummy.value = this.configObject.id;
    dummy.select();
    document.execCommand('copy');
    document.body.removeChild(dummy);

    this.snackBarService.openSnackBar('Copied');
  }

}
